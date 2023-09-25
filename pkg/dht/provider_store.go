package dht

import (
	"bufio"
	"context"
	"crypto/sha256"
	"embed"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/parsec/pkg/config"
)

//go:embed badbits.deny
var badBitsFile embed.FS

type AddProviderRecord struct {
	RemotePeer   string
	RemoteMaddrs []multiaddr.Multiaddr
	Timestamp    time.Time
	PartitionKey string
	AgentVersion string
	Fleet        string
	LocalPeer    string
	Region       string
	Multihash    string
	Preimage     string
}

type ProviderStore struct {
	conf      config.ServerConfig
	fh        *firehose.Firehose
	store     providers.ProviderStore
	batchSize int
	batchTime time.Duration
	insert    chan *AddProviderRecord
	batch     []*AddProviderRecord
	denyMap   map[string]struct{}
	host      host.Host
}

func NewProviderStore(ctx context.Context, pstore providers.ProviderStore, h host.Host, fh *firehose.Firehose, conf config.ServerConfig) (*ProviderStore, error) {
	file, err := badBitsFile.Open("badbits.deny")
	if err != nil {
		return nil, fmt.Errorf("open bad bits file: %w", err)
	}

	denyMap := map[string]struct{}{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "//") {
			continue
		}

		denyMap[line[2:]] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("closing bad bits file: %w", err)
	}

	p := &ProviderStore{
		fh:        fh,
		store:     pstore,
		batchSize: 500, // 500 is maximum
		batchTime: 30 * time.Second,
		batch:     []*AddProviderRecord{},
		insert:    make(chan *AddProviderRecord),
		conf:      conf,
		host:      h,
		denyMap:   denyMap,
	}

	go p.loop(ctx)

	return p, nil
}

func (p *ProviderStore) loop(ctx context.Context) {
	ticker := time.NewTicker(p.batchTime)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.flush()
			ticker.Reset(p.batchTime)
		case rec := <-p.insert:
			p.batch = append(p.batch, rec)

			if len(p.batch) >= p.batchSize {
				p.flush()
				ticker.Reset(p.batchTime)
			}
		}
	}
}

func (p *ProviderStore) flush() {
	logEntry := log.WithFields(log.Fields{
		"size":   len(p.batch),
		"stream": p.conf.FirehoseStream,
	})
	logEntry.Infoln("Flushing RPCs to firehose")

	if len(p.batch) == 0 {
		logEntry.Infoln("No records to flush...")
		return
	}

	putRecords := make([]*firehose.Record, len(p.batch))
	for i, addRec := range p.batch {
		dat, err := json.Marshal(addRec)
		if err != nil {
			continue
		}
		putRecords[i] = &firehose.Record{Data: dat}
	}

	logEntry.Infoln("Example record:", string(putRecords[0].Data))

	_, err := p.fh.PutRecordBatch(&firehose.PutRecordBatchInput{
		DeliveryStreamName: aws.String(p.conf.FirehoseStream),
		Records:            putRecords,
	})
	if err != nil {
		logEntry.WithError(err).Warnln("Couldn't put RPC event")
	}

	p.batch = []*AddProviderRecord{}
}

var codecs = []multicodec.Code{
	multicodec.Raw,
	multicodec.DagPb,
	multicodec.DagCbor,
	multicodec.DagJose,
}

func (p *ProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	go func() {
		_, mh, err := multihash.MHFromBytes(key)
		if err != nil {
			log.WithError(err).Warnln("failed to parse multihash from key")
			return
		}

		avStr := ""
		agentVersion, err := p.host.Peerstore().Get(prov.ID, "AgentVersion")
		if err == nil {
			if str, ok := agentVersion.(string); ok {
				avStr = str
			}
		}

		rec := &AddProviderRecord{
			RemotePeer:   prov.ID.String(),
			RemoteMaddrs: prov.Addrs,
			Timestamp:    time.Now(),
			AgentVersion: avStr,
			Multihash:    mh.String(),
			Fleet:        p.conf.Fleet,
			LocalPeer:    p.host.ID().String(),
			PartitionKey: fmt.Sprintf("%s-%s", config.Global.AWSRegion, p.conf.Fleet),
			Region:       config.Global.AWSRegion,
		}

		for _, codec := range codecs {
			v1 := cid.NewCidV1(uint64(codec), mh)
			preimage := v1.String() + "/"
			h := sha256.Sum256([]byte(preimage))
			matchStr := hex.EncodeToString(h[:])

			if _, found := p.denyMap[matchStr]; found {
				rec.Preimage = preimage
				log.WithField("preimage", preimage).Infoln("Preimage found!", prov)
				break
			}
		}

		p.insert <- rec
	}()

	return p.store.AddProvider(ctx, key, prov)
}

func (p *ProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	return p.store.GetProviders(ctx, key)
}
