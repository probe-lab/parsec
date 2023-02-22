package db

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	lru "github.com/hashicorp/golang-lru"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/parsec/pkg/config"
)

//go:embed migrations
var migrations embed.FS

type Client struct {
	ctx context.Context

	// Reference to the configuration
	conf *config.Config

	// Database handler
	dbh *sql.DB

	// protocols cache
	agentVersions *lru.Cache

	// protocols cache
	protocols *lru.Cache

	// protocols set cache
	protocolsSets *lru.Cache
}

// InitClient establishes a database connection with the provided configuration and applies any pending
// migrations
func InitClient(ctx context.Context, conf *config.Config) (*Client, error) {
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

	// Open database handle
	dbh, err := sql.Open(driverName, conf.DatabaseSourceName())
	if err != nil {
		return nil, errors.Wrap(err, "opening database")
	}

	// Ping database to verify connection.
	if err = dbh.Ping(); err != nil {
		return nil, errors.Wrap(err, "pinging database")
	}

	client := &Client{ctx: ctx, conf: conf, dbh: dbh}
	client.applyMigrations(conf, dbh)

	return client, nil
}

func (c *Client) Handle() *sql.DB {
	return c.dbh
}

func (c *Client) Close() error {
	return c.dbh.Close()
}

func (c *Client) applyMigrations(conf *config.Config, dbh *sql.DB) {
	tmpDir, err := os.MkdirTemp("", "nebula-"+conf.Version)
	if err != nil {
		log.WithError(err).WithField("pattern", "nebula-"+conf.Version).Warnln("Could not create tmp directory for migrations")
		return
	}
	defer func() {
		if err = os.RemoveAll(tmpDir); err != nil {
			log.WithError(err).WithField("tmpDir", tmpDir).Warnln("Could not clean up tmp directory")
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
			return errors.Wrap(err, "read file")
		}

		return os.WriteFile(join, data, 0o644)
	})
	if err != nil {
		log.WithError(err).Warnln("Could not create migrations files")
		return
	}

	// Apply migrations
	driver, err := postgres.WithInstance(dbh, &postgres.Config{})
	if err != nil {
		log.WithError(err).Warnln("Could not create driver instance")
		return
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+filepath.Join(tmpDir, "migrations"), conf.DatabaseName, driver)
	if err != nil {
		log.WithError(err).Warnln("Could not create migrate instance")
		return
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.WithError(err).Warnln("Couldn't apply migrations")
		return
	}
}
