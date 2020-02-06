package types

//go:generate counterfeiter . ValueGeneratorFactory

type ValueGeneratorFactory interface {
	GetGenerator(valueType string) (ValueGenerator, error)
}
