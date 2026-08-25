// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"context"
	"crypto/fips140"
	"testing"

	sgv1alpha1 "carvel.dev/secretgen-controller/pkg/apis/secretgen/v1alpha1"
)

// These tests only exercise anything meaningful when run with
// GOFIPS140=v1.0.0 GODEBUG=fips140=only, which enables strict FIPS 140-3
// enforcement (crypto/fips140.Enforced() reports it at runtime). Without
// that, they degrade to a skip so they don't add noise to a normal `go test
// ./...` run.

func TestSSHKeyReconciler_GenerateUnderFIPS140Only(t *testing.T) {
	if !fips140.Enforced() {
		t.Skip("run with GOFIPS140=v1.0.0 GODEBUG=fips140=only to exercise strict FIPS 140-3 enforcement")
	}

	r := &SSHKeyReconciler{}

	sshKey, err := r.generate(&sgv1alpha1.SSHKey{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if sshKey.PrivateKey == "" || sshKey.PublicKey == "" || sshKey.PublicKeyFingerprint == "" {
		t.Fatalf("generate returned incomplete SSHKey: %+v", sshKey)
	}
}

func TestCertificateReconciler_GenerateUnderFIPS140Only(t *testing.T) {
	if !fips140.Enforced() {
		t.Skip("run with GOFIPS140=v1.0.0 GODEBUG=fips140=only to exercise strict FIPS 140-3 enforcement")
	}

	r := &CertificateReconciler{}
	cert := &sgv1alpha1.Certificate{
		Spec: sgv1alpha1.CertificateSpec{
			CommonName: "test-ca",
			IsCA:       true,
		},
	}
	params := newCertParams(cert)

	certResp, err := r.generate(context.Background(), params, cert)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if certResp.Certificate == "" || certResp.PrivateKey == "" {
		t.Fatalf("generate returned incomplete CertResponse: %+v", certResp)
	}
}
