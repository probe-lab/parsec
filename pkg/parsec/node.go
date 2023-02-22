package parsec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/dennis-tra/parsec/pkg/util"
	"github.com/guseggert/clustertest/cluster"
	"github.com/guseggert/clustertest/cluster/basic"
	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	*basic.Node

	ctx    context.Context
	client http.Client
	host   string
	port   int
	logger *log.Logger
	fmt    log.Formatter
}

func NewNode(n *basic.Node, id string, host string, port int) (*Node, error) {
	log.Infoln("Reading parsec binary...")
	parsecBinary, err := os.ReadFile("parsec")
	if err != nil {
		return nil, fmt.Errorf("read parsec binary: %w", err)
	}

	log.Infoln("Sending parsec binary...")
	if err = n.SendFile("/parsec", bytes.NewReader(parsecBinary)); err != nil {
		return nil, fmt.Errorf("send parsec binary: %w", err)
	}

	log.Infoln("Setting parsec permissions...")
	proc, err := n.StartProc(cluster.StartProcRequest{
		Command: "chmod",
		Args:    []string{"+x", "/parsec"},
	})
	if err != nil {
		return nil, fmt.Errorf("setting parsec permissions: %w", err)
	}
	if _, err := proc.Wait(); err != nil {
		return nil, fmt.Errorf("set parsec permissions: %w", err)
	}

	newTransport := http.DefaultTransport.(*http.Transport).Clone()
	newTransport.DialContext = n.Node.Dial
	httpClient := http.Client{Transport: newTransport}

	logger := log.New()
	parsecNode := &Node{
		Node:   n,
		ctx:    context.Background(),
		client: httpClient,
		host:   host,
		port:   port,
		logger: logger,
		fmt:    logger.Formatter,
	}

	parsecNode.logger.Formatter = parsecNode
	logEntry := parsecNode.logger.WithField("nodeID", id)

	log.Infoln("Starting parsec server...")
	proc, err = n.StartProc(cluster.StartProcRequest{
		Command: "/parsec",
		Args:    []string{"server"},
		Stderr:  logEntry.WithField("fd", "stderr").Writer(),
		Stdout:  logEntry.WithField("fd", "stdout").Writer(),
	})
	if err != nil {
		return nil, fmt.Errorf("start parsec server: %w", err)
	}

	go func() {
		procRes, err := proc.Wait()
		log.WithError(err).WithField("exitCode", procRes.ExitCode).WithField("uptime", procRes.TimeMS).Infoln("Parsec server exited")
	}()

	return parsecNode, nil
}

func (n *Node) Context(ctx context.Context) *Node {
	newN := *n
	newN.Ctx = ctx
	newN.Node = newN.Node.Context(ctx)
	return &newN
}

func (n *Node) Info(c *util.Content) (*InfoResponse, error) {
	res, err := n.client.Get(fmt.Sprintf("http://%s:%d/info", n.host, n.port))
	if err != nil {
		return nil, fmt.Errorf("get info: %w", err)
	}

	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read provide response: %w", err)
	}

	info := InfoResponse{}
	if err = json.Unmarshal(dat, &info); err != nil {
		return nil, fmt.Errorf("unmarshal info response: %w", err)
	}

	return &info, nil
}

func (n *Node) Retrieve(c cid.Cid, count int) (*RetrievalResponse, error) {
	rr := &RetrieveRequest{
		Count: 1,
	}

	data, err := json.Marshal(rr)
	if err != nil {
		return nil, fmt.Errorf("marshal retrieval request: %w", err)
	}

	res, err := n.client.Post(fmt.Sprintf("http://%s:%d/retrieve/%s", n.host, n.port, c.String()), "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("post retrieval request: %w", err)
	}
	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read retrieval response: %w", err)
	}
	retrieval := RetrievalResponse{}
	if err = json.Unmarshal(dat, &retrieval); err != nil {
		return nil, fmt.Errorf("unmarshal retrieval response: %w", err)
	}

	return &retrieval, nil
}

func (n *Node) Provide(c *util.Content) (*ProvideResponse, error) {
	pr := &ProvideRequest{
		Content: c.Raw,
	}

	data, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("marshal provide request: %w", err)
	}

	res, err := n.client.Post(fmt.Sprintf("http://%s:%d/provide", n.host, n.port), "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("start provide: %w", err)
	}

	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read provide response: %w", err)
	}

	provide := ProvideResponse{}
	if err = json.Unmarshal(dat, &provide); err != nil {
		return nil, fmt.Errorf("unmarshal provide response: %w", err)
	}

	return &provide, nil
}

func (n *Node) Format(entry *log.Entry) ([]byte, error) {
	logMsg := map[string]interface{}{}
	if err := json.Unmarshal([]byte(entry.Message), &logMsg); err != nil {
		n.logger.WithError(err).Warnln("Could not unmarshal log message")
		return n.fmt.Format(entry)
	}

	for k, v := range logMsg {
		switch k {
		case "msg":
			entry.Message = v.(string)
		case "time":
			t, err := time.Parse(time.RFC3339Nano, v.(string))
			if err != nil {
				return nil, fmt.Errorf("parsing time: %w", err)
			}
			entry.Time = t
		case "level":
			l, err := log.ParseLevel(logMsg["level"].(string))
			if err != nil {
				return nil, fmt.Errorf("parsing level: %w", err)
			}
			entry.Level = l
		default:
			entry.Data[k] = v
		}
	}

	if !n.logger.IsLevelEnabled(entry.Level) {
		return nil, nil
	}

	return n.fmt.Format(entry)
}
