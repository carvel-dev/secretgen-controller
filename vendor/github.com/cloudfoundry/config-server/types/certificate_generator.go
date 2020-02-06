package types

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"crypto/sha1"

	"github.com/cloudfoundry/bosh-utils/errors"
)

type CertificateGenerator struct {
	loader CertsLoader
}

type CertResponse struct {
	Certificate string `json:"certificate" yaml:"certificate"`
	PrivateKey  string `json:"private_key" yaml:"private_key"`
	CA          string `json:"ca"          yaml:"ca"`
}

type certParams struct {
	CommonName       string   `yaml:"common_name"`
	Organization     string   `yaml:"organization"`
	Organizations    []string `yaml:"organizations"`
	AlternativeNames []string `yaml:"alternative_names"`
	IsCA             bool     `yaml:"is_ca"`
	CAName           string   `yaml:"ca"`
	ExtKeyUsage      []string `yaml:"extended_key_usage"`
	Duration         int64    `yaml:"duration"`
}

var supportedCertParameters = []string{
	"common_name",
	"organization",
	"alternative_names",
	"is_ca",
	"ca",
	"extended_key_usage",
	"duration",
}

func NewCertificateGenerator(loader CertsLoader) CertificateGenerator {
	return CertificateGenerator{loader: loader}
}

func (cfg CertificateGenerator) Generate(parameters interface{}) (interface{}, error) {
	var params certParams
	err := objToStruct(parameters, &params, supportedCertParameters)
	if err != nil {
		return nil, errors.WrapError(err, "Failed to generate certificate, parameters are invalid")
	}

	return cfg.generateCertificate(params)
}

func (cfg CertificateGenerator) bigIntHash(n *big.Int) []byte {
	h := sha1.New()
	h.Write(n.Bytes())
	return h.Sum(nil)
}

func (cfg CertificateGenerator) generateCertificate(cParams certParams) (CertResponse, error) {
	var certResponse CertResponse

	privateKey, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return certResponse, errors.WrapError(err, "Generating Key")
	}

	certTemplate, err := generateCertTemplate(cParams)
	if err != nil {
		return certResponse, err
	}

	var certificateRaw []byte
	var rootCARaw []byte
	var rootCA *x509.Certificate
	var rootPKey *rsa.PrivateKey

	if cParams.CAName != "" {
		if cfg.loader == nil {
			panic("Expected CertificateGenerator to have Loader set")
		}
		rootCA, rootPKey, err = cfg.loader.LoadCerts(cParams.CAName)
		if err != nil {
			return certResponse, errors.WrapError(err, "Loading certificates")
		}
	}

	certTemplate.SubjectKeyId = cfg.bigIntHash(privateKey.N)

	if cParams.IsCA {
		certTemplate.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign

		signingKey := privateKey
		signingCA := &certTemplate

		if cParams.CAName != "" {
			signingKey = rootPKey
			signingCA = rootCA
		}
		certTemplate.AuthorityKeyId = signingCA.SubjectKeyId

		certificateRaw, err = x509.CreateCertificate(rand.Reader, &certTemplate, signingCA, &privateKey.PublicKey, signingKey)
		if err != nil {
			return certResponse, errors.WrapError(err, "Generating CA certificate")
		}

		if cParams.CAName != "" {
			rootCARaw = rootCA.Raw
		} else {
			rootCARaw = certificateRaw
		}
	} else {
		if cParams.CAName == "" {
			return certResponse, errors.Error("Missing required CA name")
		}
		certTemplate.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature

		certTemplate.AuthorityKeyId = rootCA.SubjectKeyId

		extKeyUsages := certTemplate.ExtKeyUsage
		if len(cParams.ExtKeyUsage) != 0 {
			for _, extKeyUsage := range cParams.ExtKeyUsage {
				switch extKeyUsage {
				case "client_auth":
					extKeyUsages = append(extKeyUsages, x509.ExtKeyUsageClientAuth)
				case "server_auth":
					extKeyUsages = append(extKeyUsages, x509.ExtKeyUsageServerAuth)
				default:
					return certResponse, errors.Errorf("Unsupported extended key usage value: %s", extKeyUsage)
				}
			}
		} else {
			extKeyUsages = append(extKeyUsages, x509.ExtKeyUsageServerAuth)
		}

		certTemplate.ExtKeyUsage = extKeyUsages

		for _, altName := range cParams.AlternativeNames {
			possibleIP := net.ParseIP(altName)
			if possibleIP == nil {
				certTemplate.DNSNames = append(certTemplate.DNSNames, altName)
			} else {
				certTemplate.IPAddresses = append(certTemplate.IPAddresses, possibleIP)
			}
		}

		certificateRaw, err = x509.CreateCertificate(rand.Reader, &certTemplate, rootCA, &privateKey.PublicKey, rootPKey)
		if err != nil {
			return certResponse, errors.WrapError(err, "Generating certificate")
		}
		rootCARaw = rootCA.Raw
	}

	return generateCertResponse(privateKey, certificateRaw, rootCARaw), nil
}

func generateCertTemplate(cParams certParams) (x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return x509.Certificate{}, errors.WrapError(err, "Generating Serial Number")
	}

	now := time.Now()
	notAfter := now.Add(365 * 24 * time.Hour)

	if cParams.Duration > 0 {
		notAfter = now.Add(time.Duration(cParams.Duration*24) * time.Hour)
	}

	var organizations []string
	if cParams.Organization == "" {
		organizations = []string{"Cloud Foundry"}
	} else {
		organizations = []string{cParams.Organization}
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{"USA"},
			Organization: organizations,
			CommonName:   cParams.CommonName,
		},
		NotBefore:             now,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		IsCA:                  cParams.IsCA,
	}
	return template, nil
}

func generateCertResponse(privateKey *rsa.PrivateKey, certificateRaw, rootCARaw []byte) CertResponse {
	encodedCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateRaw})
	encodedPrivatekey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	encodedRootCACert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCARaw})

	certResponse := CertResponse{
		Certificate: string(encodedCert),
		PrivateKey:  string(encodedPrivatekey),
		CA:          string(encodedRootCACert),
	}

	return certResponse
}
