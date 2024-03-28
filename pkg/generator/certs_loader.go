// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	cfgtypes "github.com/cloudfoundry/config-server/types"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type singleCertLoader struct {
	caCertSecret *corev1.Secret
}

var _ cfgtypes.CertsLoader = singleCertLoader{}

func (l singleCertLoader) LoadCerts(_ string) (*x509.Certificate, *rsa.PrivateKey, error) {
	crt, err := l.parseCertificate(string(l.caCertSecret.Data[sgv1alpha1.CertificateSecretDefaultCertificateKey]))
	if err != nil {
		return nil, nil, err
	}

	key, err := l.parsePrivateKey(string(l.caCertSecret.Data[sgv1alpha1.CertificateSecretDefaultPrivateKeyKey]))
	if err != nil {
		return nil, nil, err
	}

	return crt, key, nil
}

func (singleCertLoader) parseCertificate(data string) (*x509.Certificate, error) {
	cpb, _ := pem.Decode([]byte(data))
	if cpb == nil {
		return nil, fmt.Errorf("Certificate did not contain PEM formatted block")
	}

	crt, err := x509.ParseCertificate(cpb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Parsing certificate: %s", err)
	}

	return crt, nil
}

func (singleCertLoader) parsePrivateKey(data string) (*rsa.PrivateKey, error) {
	kpb, _ := pem.Decode([]byte(data))
	if kpb == nil {
		return nil, fmt.Errorf("Private key did not contain PEM formatted block")
	}

	key, err := x509.ParsePKCS1PrivateKey(kpb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Parsing private key: %s", err)
	}

	return key, nil
}
