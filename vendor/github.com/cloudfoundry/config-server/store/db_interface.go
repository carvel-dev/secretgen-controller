package store

import "database/sql"

//go:generate counterfeiter . IDb
//go:generate counterfeiter database/sql.Result

type IDb interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (IRows, error)
	QueryRow(query string, args ...interface{}) IRow
	SetMaxOpenConns(n int)
	SetMaxIdleConns(n int)
	Close()
}
