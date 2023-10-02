package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/parsec/pkg/config"
)

type Config struct {
	Fleet     string
	DBNodeID  int
	Region    string
	Stream    string
	BatchSize int
	BatchTime time.Duration
	Badbits   string
}

type Event struct {
	EventType    string
	Timestamp    time.Time
	RemotePeer   string
	RemoteMaddrs []multiaddr.Multiaddr
	PartitionKey string
	AgentVersion string
	DBNodeID     int
	Fleet        string
	LocalPeer    string
	Region       string
	Payload      json.RawMessage
}

type Client struct {
	host   host.Host
	fh     *firehose.Firehose
	conf   *Config
	insert chan *Event
	batch  []*Event
}

func NewClient(ctx context.Context, conf *Config) (*Client, error) {
	log.Infoln("Initializing firehose stream")
	fh, err := initStream(conf.Region, conf.Stream)
	if err != nil {
		return nil, err
	}

	p := &Client{
		fh:     fh,
		conf:   conf,
		insert: make(chan *Event),
		batch:  make([]*Event, 0, conf.BatchSize),
	}

	go p.loop(ctx)

	return p, nil
}

func (c *Client) SetHost(h host.Host) {
	c.host = h
}

func initStream(region, stream string) (*firehose.Firehose, error) {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("new aws session: %w", err)
	}
	fh := firehose.New(awsSession)

	streamName := aws.String(stream)

	log.Infoln("Checking firehose stream permissions")
	_, err = fh.DescribeDeliveryStream(&firehose.DescribeDeliveryStreamInput{
		DeliveryStreamName: streamName,
	})
	if err != nil {
		return nil, fmt.Errorf("describing firehose stream: %w", err)
	}

	return fh, nil
}

func (c *Client) loop(ctx context.Context) {
	ticker := time.NewTicker(c.conf.BatchTime)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.flush()
			ticker.Reset(c.conf.BatchTime)
		case rec := <-c.insert:
			c.batch = append(c.batch, rec)

			if len(c.batch) >= c.conf.BatchSize {
				c.flush()
				ticker.Reset(c.conf.BatchTime)
			}
		}
	}
}

func (c *Client) flush() {
	logEntry := log.WithFields(log.Fields{
		"size":   len(c.batch),
		"stream": c.conf.Stream,
	})
	logEntry.Infoln("Flushing RPCs to firehose")

	if len(c.batch) == 0 {
		logEntry.Infoln("No records to flush...")
		return
	}

	putRecords := make([]*firehose.Record, len(c.batch))
	for i, addRec := range c.batch {
		dat, err := json.Marshal(addRec)
		if err != nil {
			continue
		}
		putRecords[i] = &firehose.Record{Data: dat}
	}

	_, err := c.fh.PutRecordBatch(&firehose.PutRecordBatchInput{
		DeliveryStreamName: aws.String(c.conf.Stream),
		Records:            putRecords,
	})
	if err != nil {
		logEntry.WithError(err).Warnln("Couldn't put RPC event")
	} else {
		logEntry.Infof("Flushed %d records!\n", len(putRecords))
	}

	c.batch = make([]*Event, c.conf.BatchSize)
}

func (c *Client) Submit(evtType string, remotePeer peer.ID, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	avStr := ""
	agentVersion, err := c.host.Peerstore().Get(remotePeer, "AgentVersion")
	if err == nil {
		if str, ok := agentVersion.(string); ok {
			avStr = str
		}
	}

	evt := &Event{
		EventType:    evtType,
		Timestamp:    time.Now(),
		RemotePeer:   remotePeer.String(),
		RemoteMaddrs: c.host.Peerstore().Addrs(remotePeer),
		PartitionKey: fmt.Sprintf("%s-%s", config.Global.AWSRegion, c.conf.Fleet),
		AgentVersion: avStr,
		DBNodeID:     c.conf.DBNodeID,
		Fleet:        c.conf.Fleet,
		LocalPeer:    c.host.ID().String(),
		Region:       config.Global.AWSRegion,
		Payload:      data,
	}

	c.insert <- evt

	return nil
}
