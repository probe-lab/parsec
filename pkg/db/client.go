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
	"time"

	"github.com/dennis-tra/parsec/pkg/server"

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

	"github.com/dennis-tra/parsec/pkg/models"
)

type Client interface {
	InsertRun(ctx context.Context) (*models.Run, error)
	InsertNode(ctx context.Context, dbRunID int, peerID peer.ID, region string, it string, bi *debug.BuildInfo) (*models.Node, error)
	InsertRetrieval(ctx context.Context, dbNodeID int, retrieval *server.RetrievalResponse) (*models.Retrieval, error)
	InsertProvide(ctx context.Context, dbNodeID int, provide *server.ProvideResponse) (*models.Provide, error)
	Close() error
}

//go:embed migrations
var migrations embed.FS

type DBClient struct {
	// Database handle
	handle *sql.DB
}

// InitDBClient establishes a database connection with the provided configuration and applies any pending
// migrations
func InitDBClient(ctx context.Context, host string, port int, name string, user string, password string, ssl string) (Client, error) {
	log.WithFields(log.Fields{
		"host": host,
		"port": port,
		"name": name,
		"user": user,
		"ssl":  ssl,
	}).Infoln("Initializing database client")

	driverName, err := ocsql.Register("postgres")
	if err != nil {
		return nil, errors.Wrap(err, "register ocsql")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		host, port, name, user, password, ssl,
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

	client := &DBClient{handle: db}

	return client, client.applyMigrations(db, name)
}

func (c *DBClient) Close() error {
	return c.handle.Close()
}

func (c *DBClient) applyMigrations(db *sql.DB, name string) error {
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
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create driver instance: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+filepath.Join(tmpDir, "migrations"), name, driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func (c *DBClient) InsertRun(ctx context.Context) (*models.Run, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, fmt.Errorf("read build info")
	}

	biData, err := json.Marshal(bi)
	if err != nil {
		return nil, fmt.Errorf("marshal build info data: %w", err)
	}

	dbRun := &models.Run{
		Dependencies: biData,
		StartedAt:    time.Now(),
	}

	return dbRun, dbRun.Insert(ctx, c.handle, boil.Infer())
}

func (c *DBClient) InsertNode(ctx context.Context, dbRunID int, peerID peer.ID, region string, it string, bi *debug.BuildInfo) (*models.Node, error) {
	biData, err := json.Marshal(bi)
	if err != nil {
		return nil, fmt.Errorf("marshal build info data: %w", err)
	}

	n := &models.Node{
		RunID:        dbRunID,
		PeerID:       peerID.String(),
		Region:       region,
		InstanceType: it,
		Dependencies: biData,
	}

	return n, n.Insert(ctx, c.handle, boil.Infer())
}

func (c *DBClient) InsertRetrieval(ctx context.Context, dbNodeID int, retrieval *server.RetrievalResponse) (*models.Retrieval, error) {
	r := &models.Retrieval{
		Cid:      retrieval.CID,
		NodeID:   dbNodeID,
		Duration: retrieval.Duration.Seconds(),
		RTSize:   retrieval.RoutingTableSize,
		Error:    null.NewString(retrieval.Error, retrieval.Error != ""),
	}

	return r, r.Insert(ctx, c.handle, boil.Infer())
}

func (c *DBClient) InsertProvide(ctx context.Context, dbNodeID int, provide *server.ProvideResponse) (*models.Provide, error) {
	p := &models.Provide{
		Cid:      provide.CID,
		NodeID:   dbNodeID,
		Duration: provide.Duration.Seconds(),
		RTSize:   provide.RoutingTableSize,
		Error:    null.NewString(provide.Error, provide.Error != ""),
	}

	return p, p.Insert(ctx, c.handle, boil.Infer())
}

type DummyClient struct{}

func NewDummyClient() Client {
	return &DummyClient{}
}

func (d *DummyClient) InsertRun(ctx context.Context) (*models.Run, error) {
	return &models.Run{ID: 1}, nil
}

func (d *DummyClient) InsertNode(ctx context.Context, dbRunID int, peerID peer.ID, region string, it string, bi *debug.BuildInfo) (*models.Node, error) {
	return &models.Node{RunID: dbRunID, Region: region, InstanceType: it, PeerID: peerID.String()}, nil
}

func (d *DummyClient) InsertRetrieval(ctx context.Context, dbNodeID int, retrieval *server.RetrievalResponse) (*models.Retrieval, error) {
	return &models.Retrieval{NodeID: dbNodeID}, nil
}

func (d *DummyClient) InsertProvide(ctx context.Context, dbNodeID int, provide *server.ProvideResponse) (*models.Provide, error) {
	return &models.Provide{NodeID: dbNodeID}, nil
}

func (d *DummyClient) Close() error {
	return nil
}
