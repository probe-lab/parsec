package db

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/models"
)

type Client interface {
	InsertScheduler(ctx context.Context, fleets []string) (*models.Scheduler, error)
	InsertNode(ctx context.Context, peerID peer.ID, conf config.ServerConfig) (*models.Node, error)
	GetNodes(ctx context.Context, fleets []string) (models.NodeSlice, error)
	InsertRetrieval(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Retrieval, error)
	InsertProvide(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Provide, error)
	UpdateHeartbeat(ctx context.Context, dbNode *models.Node) error
	UpdateOfflineSince(ctx context.Context, dbNode *models.Node) error
	Close() error
}

//go:embed migrations
var migrations embed.FS

type DBClient struct {
	// Database handle
	handle *sql.DB
	conf   config.GlobalConfig
}

var _ Client = (*DBClient)(nil)

// InitDBClient establishes a database connection with the provided configuration and applies any pending
// migrations
func InitDBClient(ctx context.Context, conf config.GlobalConfig) (Client, error) {
	log.WithFields(log.Fields{
		"host": conf.DatabaseHost,
		"port": conf.DatabasePort,
		"name": conf.DatabaseName,
		"user": conf.DatabaseUser,
		"ssl":  conf.DatabaseSSLMode,
	}).Infoln("Initializing database client")

	driverName, err := ocsql.Register("postgres")
	if err != nil {
		return nil, errors.Wrap(err, "register ocsql")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		conf.DatabaseHost, conf.DatabasePort, conf.DatabaseName, conf.DatabaseUser, conf.DatabasePassword, conf.DatabaseSSLMode,
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

	client := &DBClient{
		handle: db,
		conf:   conf,
	}

	return client, client.applyMigrations()
}

func (c *DBClient) Close() error {
	return c.handle.Close()
}

func (c *DBClient) applyMigrations() error {
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

	m, err := migrate.NewWithDatabaseInstance("file://"+filepath.Join(tmpDir, "migrations"), c.conf.DatabaseName, driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func (c *DBClient) InsertScheduler(ctx context.Context, fleets []string) (*models.Scheduler, error) {
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

func (c *DBClient) InsertNode(ctx context.Context, peerID peer.ID, conf config.ServerConfig) (*models.Node, error) {
	sp, err := c.conf.ServerProcess()
	if err != nil {
		return nil, fmt.Errorf("server process: %w", err)
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, fmt.Errorf("read build info error")
	}

	biData, err := json.Marshal(bi)
	if err != nil {
		return nil, fmt.Errorf("marshal build info data: %w", err)
	}

	n := &models.Node{
		CPU:          int(sp.CPU),
		Memory:       int(sp.Memory),
		PeerID:       peerID.String(),
		Region:       c.conf.AWSRegion,
		CMD:          strings.Join(os.Args, " "),
		Dependencies: biData,
		IPAddress:    sp.PrivateIP.String(),
		Fleet:        conf.Fleet,
		ServerPort:   int16(conf.ServerPort),
		PeerPort:     int16(conf.PeerPort),
	}

	return n, n.Insert(ctx, c.handle, boil.Infer())
}

func (c *DBClient) GetNodes(ctx context.Context, fleets []string) (models.NodeSlice, error) {
	wheres := []qm.QueryMod{
		models.NodeWhere.OfflineSince.IsNull(),
		models.NodeWhere.LastHeartbeat.GTE(null.TimeFrom(time.Now().Add(-2 * time.Minute))),
		models.NodeWhere.Fleet.IN(fleets),
	}

	return models.Nodes(wheres...).All(ctx, c.handle)
}

func (c *DBClient) UpdateHeartbeat(ctx context.Context, dbNode *models.Node) error {
	log.Debugln("Update heartbeat", dbNode.ID)
	dbNode.LastHeartbeat = null.TimeFrom(time.Now())
	dbNode.OfflineSince = null.NewTime(time.Now(), false)

	_, err := dbNode.Update(ctx, c.handle, boil.Infer())

	return err
}

func (c *DBClient) UpdateOfflineSince(ctx context.Context, dbNode *models.Node) error {
	log.Debugln("Update node offline", dbNode.ID)
	dbNode.OfflineSince = null.TimeFrom(time.Now())
	_, err := dbNode.Update(ctx, c.handle, boil.Infer())
	return err
}

func (c *DBClient) InsertRetrieval(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Retrieval, error) {
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

func (c *DBClient) InsertProvide(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Provide, error) {
	p := &models.Provide{
		Cid:         cid,
		NodeID:      dbNodeID,
		Duration:    duration,
		RTSize:      rtSize,
		SchedulerID: schedulerID,
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

func (d *DummyClient) InsertNode(ctx context.Context, peerID peer.ID, conf config.ServerConfig) (*models.Node, error) {
	return &models.Node{Region: "dummy", PeerID: peerID.String()}, nil
}

func (d *DummyClient) InsertRetrieval(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Retrieval, error) {
	return &models.Retrieval{NodeID: dbNodeID}, nil
}

func (d *DummyClient) InsertProvide(ctx context.Context, dbNodeID int, cid string, duration float64, rtSize int, errStr string, schedulerID int) (*models.Provide, error) {
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
