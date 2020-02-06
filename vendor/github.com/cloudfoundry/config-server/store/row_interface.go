package store

//go:generate counterfeiter . IRow

type IRow interface {
	Scan(dest ...interface{}) error
}
