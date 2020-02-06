package store

//go:generate counterfeiter . DbProvider

type DbProvider interface {
	Db() (IDb, error)
}
