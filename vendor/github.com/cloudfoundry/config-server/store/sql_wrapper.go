package store

import (
	"github.com/BurntSushi/migration"
)

type SQLWrapper struct {
}

func NewSQLWrapper() SQLWrapper {
	return SQLWrapper{}
}

func (w SQLWrapper) OpenWith(
	driverName,
	dataSourceName string,
	migrations []migration.Migrator,
	versionGet migration.GetVersion,
	versionSet migration.SetVersion,
) (IDb, error) {
	db, err := migration.OpenWith(driverName, dataSourceName, migrations, versionGet, versionSet)
	return NewDbWrapper(db), err
}
