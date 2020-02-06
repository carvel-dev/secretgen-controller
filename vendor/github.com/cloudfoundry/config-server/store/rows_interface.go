package store

//go:generate counterfeiter . IRows

type IRows interface {
	Next() bool
	Close() error
	Scan(dest ...interface{}) error
}
