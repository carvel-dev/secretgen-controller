package server

//go:generate counterfeiter . TokenValidator
//go:generate counterfeiter net/http.Handler

type TokenValidator interface {
	Validate(token string) error
}
