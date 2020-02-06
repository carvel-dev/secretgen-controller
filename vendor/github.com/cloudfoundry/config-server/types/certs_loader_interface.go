package types

import (
	"crypto/rsa"
	"crypto/x509"
)

//go:generate counterfeiter . CertsLoader

type CertsLoader interface {
	LoadCerts(string) (*x509.Certificate, *rsa.PrivateKey, error)
}
