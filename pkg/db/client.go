package db

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"

	"github.com/probe-lab/parsec/pkg/models"
)

type Client interface {
	InsertScheduler(ctx context.Context, fleets []string) (*models.Scheduler, error)
	InsertNode(ctx context.Context, model *NodeModel) (*models.Node, error)
	GetNodes(ctx context.Context, fleets []string) (models.NodeSlice, error)
	InsertRetrieval(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Retrieval, error)
	InsertProvide(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int, step string) (*models.Provide, error)
	UpdateHeartbeat(ctx context.Context, dbNode *models.Node) error
	UpdateOfflineSince(ctx context.Context, dbNode *models.Node) error
	Close() error
}

//go:embed migrations
var migrations embed.FS

type PGClientConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type PGClient struct {
	// Database handle
	handle *sql.DB
	conf   *PGClientConfig
}

var _ Client = (*PGClient)(nil)

// InitPGClient establishes a database connection with the provided configuration and applies any pending
// migrations
func InitPGClient(ctx context.Context, conf *PGClientConfig) (Client, error) {
	log.WithFields(log.Fields{
		"host": conf.Host,
		"port": conf.Port,
		"name": conf.Name,
		"user": conf.User,
		"ssl":  conf.SSLMode,
	}).Infoln("Initializing database client")

	driverName, err := ocsql.Register("postgres")
	if err != nil {
		return nil, fmt.Errorf("register ocsql: %w", err)
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		conf.Host, conf.Port, conf.Name, conf.User, conf.Password, conf.SSLMode,
	)

	// Open database handle
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Ping database to verify connection.
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	client := &PGClient{
		handle: db,
		conf:   conf,
	}

	return client, client.applyMigrations()
}

func (c *PGClient) Close() error {
	return c.handle.Close()
}

func (c *PGClient) applyMigrations() error {
	tmpDir, err := os.MkdirTemp("", "parsec")
	if err != nil {
		return fmt.Errorf("create migrations tmp dir: %w", err)
	}
	defer func() {
		if err = os.RemoveAll(tmpDir); err != nil {
			log.WithField("tmpDir", tmpDir).WithError(err).Warnln("Could not clean up tmp directory")
		}
	}()
	log.WithField("dir", tmpDir).Debugln("Created temporary directory")

	err = fs.WalkDir(migrations, ".", func(path string, d fs.DirEntry, err error) error {
		join := filepath.Join(tmpDir, path)
		if d.IsDir() {
			return os.MkdirAll(join, 0o755)
		}

		data, err := migrations.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		return os.WriteFile(join, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("create migrations files: %w", err)
	}

	// Apply migrations
	driver, err := postgres.WithInstance(c.handle, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create driver instance: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+filepath.Join(tmpDir, "migrations"), c.conf.Name, driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func (c *PGClient) InsertScheduler(ctx context.Context, fleets []string) (*models.Scheduler, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, fmt.Errorf("read build info error")
	}

	biData, err := json.Marshal(bi)
	if err != nil {
		return nil, fmt.Errorf("marshal build info data: %w", err)
	}

	s := &models.Scheduler{
		Fleets:       fleets,
		Dependencies: biData,
	}

	return s, s.Insert(ctx, c.handle, boil.Infer())
}

type NodeModel struct {
	PeerID     peer.ID
	AWSRegion  string
	Fleet      string
	ServerPort int
	PeerPort   int
	CPU        int64
	Memory     int64
	PrivateIP  net.IP
}

func (c *PGClient) InsertNode(ctx context.Context, model *NodeModel) (*models.Node, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, fmt.Errorf("read build info error")
	}

	biData, err := json.Marshal(bi)
	if err != nil {
		return nil, fmt.Errorf("marshal build info data: %w", err)
	}

	n := &models.Node{
		CPU:          int(model.CPU),
		Memory:       int(model.Memory),
		PeerID:       model.PeerID.String(),
		Region:       model.AWSRegion,
		CMD:          strings.Join(os.Args, " "),
		Dependencies: biData,
		IPAddress:    model.PrivateIP.String(),
		Fleet:        model.Fleet,
		ServerPort:   int16(model.ServerPort),
		PeerPort:     int16(model.PeerPort),
	}

	return n, n.Insert(ctx, c.handle, boil.Infer())
}

func (c *PGClient) GetNodes(ctx context.Context, fleets []string) (models.NodeSlice, error) {
	wheres := []qm.QueryMod{
		models.NodeWhere.OfflineSince.IsNull(),
		models.NodeWhere.LastHeartbeat.GTE(null.TimeFrom(time.Now().Add(-2 * time.Minute))),
		models.NodeWhere.Fleet.IN(fleets),
	}

	return models.Nodes(wheres...).All(ctx, c.handle)
}

func (c *PGClient) UpdateHeartbeat(ctx context.Context, dbNode *models.Node) error {
	log.Debugln("Update heartbeat", dbNode.ID)
	dbNode.LastHeartbeat = null.TimeFrom(time.Now())
	dbNode.OfflineSince = null.NewTime(time.Now(), false)

	_, err := dbNode.Update(ctx, c.handle, boil.Infer())

	return err
}

func (c *PGClient) UpdateOfflineSince(ctx context.Context, dbNode *models.Node) error {
	log.Debugln("Update node offline", dbNode.ID)
	dbNode.OfflineSince = null.TimeFrom(time.Now())
	_, err := dbNode.Update(ctx, c.handle, boil.Infer())
	return err
}

func (c *PGClient) InsertRetrieval(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Retrieval, error) {
	r := &models.Retrieval{
		Cid:         cid,
		NodeID:      dbNodeID,
		Duration:    duration,
		RTSize:      rtSize,
		SchedulerID: schedulerID,
		Error:       null.NewString(errStr, errStr != ""),
	}

	return r, r.Insert(ctx, c.handle, boil.Infer())
}

func (c *PGClient) InsertProvide(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int, step string) (*models.Provide, error) {
	p := &models.Provide{
		Cid:         cid,
		NodeID:      dbNodeID,
		Duration:    duration,
		RTSize:      rtSize,
		SchedulerID: schedulerID,
		Step:        step,
		Error:       null.NewString(errStr, errStr != ""),
	}

	return p, p.Insert(ctx, c.handle, boil.Infer())
}

type DummyClient struct{}

func (d *DummyClient) GetNodes(ctx context.Context, fleets []string) (models.NodeSlice, error) {
	return []*models.Node{}, nil
}

func NewDummyClient() Client {
	return &DummyClient{}
}

func (d *DummyClient) InsertScheduler(ctx context.Context, fleets []string) (*models.Scheduler, error) {
	return &models.Scheduler{Fleets: fleets}, nil
}

func (d *DummyClient) InsertNode(ctx context.Context, model *NodeModel) (*models.Node, error) {
	return &models.Node{Region: "dummy", PeerID: model.PeerID.String()}, nil
}

func (d *DummyClient) InsertRetrieval(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Retrieval, error) {
	return &models.Retrieval{NodeID: dbNodeID}, nil
}

func (d *DummyClient) InsertProvide(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int, step string) (*models.Provide, error) {
	return &models.Provide{NodeID: dbNodeID}, nil
}

func (d *DummyClient) Close() error {
	return nil
}

func (d *DummyClient) UpdateHeartbeat(ctx context.Context, dbNode *models.Node) error {
	return nil
}

func (d *DummyClient) UpdateOfflineSince(ctx context.Context, dbNode *models.Node) error {
	return nil
}
