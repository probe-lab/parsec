package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

type Submitter interface {
	Submit(evtType string, localID peer.ID, addrInfo peer.AddrInfo, agentVersion string, payload any) error
}

type Config struct {
	Fleet     string
	AWSRegion string
	Stream    string
	BatchSize int
	BatchTime time.Duration
}

type event struct {
	EventType    string
	Timestamp    time.Time
	RemotePeer   string
	RemoteMaddrs []multiaddr.Multiaddr
	PartitionKey string
	AgentVersion string
	Fleet        string
	LocalPeer    string
	Region       string
	Payload      json.RawMessage
}

type Client struct {
	fh     *firehose.Firehose
	conf   *Config
	insert chan *event
	batch  []*event
}

var _ Submitter = (*Client)(nil)

func NewClient(ctx context.Context, conf *Config) (Submitter, error) {
	if conf.Stream != "" && conf.AWSRegion != "" {
		log.WithField("stream", conf.Stream).WithField("region", conf.AWSRegion).Infoln("Using Firehose to track connection events")
		fh, err := initStream(conf.AWSRegion, conf.Stream)
		if err != nil {
			return nil, fmt.Errorf("new firehose stream: %w", err)
		}

		c := &Client{
			fh:     fh,
			conf:   conf,
			insert: make(chan *event),
			batch:  []*event{},
		}

		go c.loop(ctx)

		return c, nil
	} else {
		log.Infoln("Not using Firehose to track connection events")
		return &NoopClient{}, nil
	}
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

	c.batch = []*event{}
}

func (c *Client) Submit(evtType string, localID peer.ID, addrInfo peer.AddrInfo, agentVersion string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	evt := &event{
		EventType:    evtType,
		Timestamp:    time.Now(),
		RemotePeer:   addrInfo.ID.String(),
		RemoteMaddrs: addrInfo.Addrs,
		PartitionKey: fmt.Sprintf("%s-%s", c.conf.AWSRegion, c.conf.Fleet),
		AgentVersion: agentVersion,
		Fleet:        c.conf.Fleet,
		LocalPeer:    localID.String(),
		Region:       c.conf.AWSRegion,
		Payload:      data,
	}

	c.insert <- evt

	return nil
}

type NoopClient struct{}

func (n *NoopClient) Submit(evtType string, localID peer.ID, addrInfo peer.AddrInfo, agentVersion string, payload any) error {
	return nil
}

var _ Submitter = (*NoopClient)(nil)
