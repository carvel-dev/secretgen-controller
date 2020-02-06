package store

import "github.com/BurntSushi/migration"

//go:generate counterfeiter . ISql

type ISql interface {
	OpenWith(
		driverName,
		dataSourceName string,
		migrations []migration.Migrator,
		getVersion migration.GetVersion,
		setVersion migration.SetVersion,
	) (IDb, error)
}
