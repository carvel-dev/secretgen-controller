// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/rest"

	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/generator"
)

func Test_RestConfig(t *testing.T) {
	type test struct {
		name            string
		localKubeconfig rest.Config
		saConfig        rest.Config
	}

	tests := []test{
		{
			name: "uses CAData if it is present (such as in local kubeconfig)",
			localKubeconfig: rest.Config{
				Host: "https://www.host.com",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte("-------FAKE LOCAL CERT------"),
				},
			},
			saConfig: rest.Config{
				Host: "https://www.host.com",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte("-------FAKE LOCAL CERT------"),
				},
				BearerToken: "my-token",
			},
		},
		{
			name: "parses CAFile as CAData if it is present (such as in cluster)",
			localKubeconfig: rest.Config{
				Host: "https://www.host.com",
				TLSClientConfig: rest.TLSClientConfig{
					CAFile: "/var/in/cluster/file.crt",
				},
			},
			saConfig: rest.Config{
				Host: "https://www.host.com",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte("-------FAKE LOCAL CERT------"),
				},
				BearerToken: "my-token",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := &manager{token: "my-token"}
			configGetter := &configGetter{config: tc.localKubeconfig}
			caCertGetter := &caCertGetter{caCert: []byte("-------FAKE LOCAL CERT------")}
			saLoader := generator.NewServiceAccountLoader(manager, configGetter, caCertGetter)
			cfg, err := saLoader.RestConfig(context.Background(), "test-name", "test-namespace")
			require.NoError(t, err)

			assert.Equal(t, caCertGetter.caCert, cfg.CAData)
			assert.Equal(t, "test-name", manager.saName)
			assert.Equal(t, "test-namespace", manager.saNamespace)
		})
	}
}

type manager struct {
	token       string
	saName      string
	saNamespace string
}

func (m *manager) GetServiceAccountToken(ctx context.Context, namespace, name string, tr *authv1.TokenRequest) (*authv1.TokenRequest, error) {
	m.saName = name
	m.saNamespace = namespace
	return &authv1.TokenRequest{
		Status: authv1.TokenRequestStatus{
			Token: m.token,
		},
	}, nil
}

type configGetter struct {
	config rest.Config
}

func (g *configGetter) GetConfig() (*rest.Config, error) {
	return &g.config, nil
}

type caCertGetter struct {
	caCert []byte
	cfg    *rest.Config
}

func (r *caCertGetter) GetCACert(cfg *rest.Config) ([]byte, error) {
	r.cfg = cfg
	return r.caCert, nil
}
