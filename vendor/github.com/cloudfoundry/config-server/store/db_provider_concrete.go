package store

import (
	"fmt"

	"github.com/BurntSushi/migration"
	"github.com/cloudfoundry/bosh-utils/errors"

	// blank import to load database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"os"
	"os/signal"
	"syscall"

	"github.com/cloudfoundry/config-server/config"
	"github.com/cloudfoundry/config-server/store/db_migrations"
)

type concreteDbProvider struct {
	config config.DBConfig
	sql    ISql
	db     IDb
}

func NewConcreteDbProvider(sql ISql, config config.DBConfig) (DbProvider, error) {
	connectionString, err := connectionString(config)
	if err != nil {
		return nil, errors.WrapError(err, "Failed to generate DB connection string")
	}

	versionGet := getVersionImpl(config)
	versionSet := setVersionImpl(config)

	db, err := sql.OpenWith(
		config.Adapter,
		connectionString,
		db_migrations.GetMigrations(config.Adapter),
		versionGet,
		versionSet,
	)
	if err != nil {
		return nil, errors.WrapError(err, "Failed to open connection to DB")
	}
	go closeDBOnSignal(db)

	db.SetMaxOpenConns(config.ConnectionOptions.MaxOpenConnections)
	db.SetMaxIdleConns(config.ConnectionOptions.MaxIdleConnections)

	provider := concreteDbProvider{db: db}
	return provider, err
}

func (p concreteDbProvider) Db() (IDb, error) {
	if p.db == nil {
		return nil, errors.Error("Database not initialized")
	}
	return p.db, nil
}

func connectionString(config config.DBConfig) (string, error) {

	var connectionString string
	var err error

	switch config.Adapter {
	case "postgres":
		connectionString = fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
			config.User, config.Password, config.Name)
	case "mysql":
		connectionString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			config.User, config.Password, config.Host, config.Port, config.Name)
	default:
		err = errors.Errorf("Unsupported adapter: %s", config.Adapter)
	}

	return connectionString, err
}

func getVersionImpl(config config.DBConfig) migration.GetVersion {
	switch config.Adapter {
	case "mysql":
		return MysqlGetVersion
	default:
		return migration.DefaultGetVersion
	}
}

func setVersionImpl(config config.DBConfig) migration.SetVersion {
	switch config.Adapter {
	case "mysql":
		return MysqlSetVersion
	default:
		return migration.DefaultSetVersion
	}
}

func closeDBOnSignal(db IDb) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	_ = <-c
	fmt.Printf("Shutting down DB connection")
	db.Close()
	os.Exit(1)
}

func MysqlGetVersion(tx migration.LimitedTx) (int, error) {
	v, err := getVersion(tx)
	if err != nil {
		if err := createVersionTable(tx); err != nil {
			return 0, err
		}
		return getVersion(tx)
	}
	return v, nil
}

func MysqlSetVersion(tx migration.LimitedTx, version int) error {
	if err := setVersion(tx, version); err != nil {
		if err := createVersionTable(tx); err != nil {
			return err
		}
		return setVersion(tx, version)
	}
	return nil
}

func getVersion(tx migration.LimitedTx) (int, error) {
	var version int
	r := tx.QueryRow("SELECT version FROM migration_version")
	if err := r.Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func setVersion(tx migration.LimitedTx, version int) error {
	_, err := tx.Exec("UPDATE migration_version SET version = ?", version)
	return err
}

func createVersionTable(tx migration.LimitedTx) error {
	_, err := tx.Exec(`CREATE TABLE migration_version ( version INTEGER);`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO migration_version (version) VALUES (0)`)
	return err
}
