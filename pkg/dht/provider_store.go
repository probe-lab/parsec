package dht

import (
	"bufio"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/multiformats/go-multicodec"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/parsec/pkg/firehose"
)

type AddProviderRecord struct {
	Multihash string
	Preimage  string
}

type ProviderStore struct {
	fhClient *firehose.Client
	store    providers.ProviderStore
	denyMap  map[string]struct{}
}

func NewProviderStore(ctx context.Context, pstore providers.ProviderStore, fhClient *firehose.Client, badbits string) (*ProviderStore, error) {
	denyMap, err := loadBadbits(badbits)
	if err != nil {
		return nil, fmt.Errorf("load bad bits: %w", err)
	}

	p := &ProviderStore{
		fhClient: fhClient,
		store:    pstore,
		denyMap:  denyMap,
	}

	return p, nil
}

func loadBadbits(filename string) (map[string]struct{}, error) {
	log.Infoln("Parsing Badbits file")
	file, err := os.Open(filename)
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

	return denyMap, nil
}

func (p *ProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	go func() {
		_, mh, err := multihash.MHFromBytes(key)
		if err != nil {
			log.WithError(err).Warnln("failed to parse multihash from key")
			return
		}

		rec := &AddProviderRecord{
			Multihash: mh.String(),
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
		if err := p.fhClient.Submit("add_provider", prov.ID, rec); err != nil {
			log.WithError(err).Warnln("Couldn't submit add_provider event")
		}
	}()

	return p.store.AddProvider(ctx, key, prov)
}

func (p *ProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	return p.store.GetProviders(ctx, key)
}

var codecs = []multicodec.Code{
	multicodec.Raw,
	multicodec.DagPb,
	multicodec.DagCbor,
	multicodec.DagJose,
}
