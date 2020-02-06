package types

//go:generate counterfeiter . ValueGenerator

type ValueGenerator interface {
	Generate(interface{}) (interface{}, error)
}
