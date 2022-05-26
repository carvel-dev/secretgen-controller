// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"context"
	"fmt"
	"os"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// TokenManager handles getting a valid token for a given ServiceAccount.
type TokenManager interface {
	GetServiceAccountToken(ctx context.Context, namespace, name string, tr *authv1.TokenRequest) (*authv1.TokenRequest, error)
}

// kubeconfigGetter encapsulates the logic for getting a kubeconfig, either within or outside a cluster.
type kubeconfigGetter interface {
	GetConfig() (*rest.Config, error)
}

// caCertGetter encapsulates the logic for extracting the CA Data from a Kubernetes config.
type caCertGetter interface {
	GetCACert(cfg *rest.Config) ([]byte, error)
}

// ServiceAccountLoader allows the construction of a k8s client from a Service Account
type ServiceAccountLoader struct {
	// Ensures a valid token for a ServiceAccount is available.
	tokenManager TokenManager

	kubeconfigGetter kubeconfigGetter
	caCertGetter     caCertGetter
}

// NewServiceAccountLoader creates a new ServiceAccountLoader
func NewServiceAccountLoader(manager TokenManager, getter kubeconfigGetter, caCertGetter caCertGetter) *ServiceAccountLoader {
	return &ServiceAccountLoader{tokenManager: manager, kubeconfigGetter: getter, caCertGetter: caCertGetter}
}

// Client returns a new k8s client for a Service Account
func (s *ServiceAccountLoader) Client(ctx context.Context, saName, saNamespace string) (client.Client, error) {
	config, err := s.RestConfig(ctx, saName, saNamespace)
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{})
}

// RestConfig get all the necessary parts (token, host, CA data) for a ServiceAccount and create a Kubernetes config for it
func (s *ServiceAccountLoader) RestConfig(ctx context.Context, saName, saNamespace string) (*rest.Config, error) {
	cfg, err := s.kubeconfigGetter.GetConfig()
	if err != nil {
		return nil, err
	}

	expiration := int64(time.Hour.Seconds())
	tokenRequest, err := s.tokenManager.GetServiceAccountToken(ctx, saNamespace, saName, &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: &expiration,
		},
	})
	if err != nil {
		return nil, err
	}

	caData, err := s.caCertGetter.GetCACert(cfg)
	if err != nil {
		return nil, err
	}

	return &rest.Config{
		Host: cfg.Host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
		BearerToken: tokenRequest.Status.Token,
	}, nil
}

var _ kubeconfigGetter = &KubeconfigGetter{}

// KubeconfigGetter can get the local Kubeconfig.
type KubeconfigGetter struct{}

// GetConfig gets the local config, be that in or out of a cluster.
func (g *KubeconfigGetter) GetConfig() (*rest.Config, error) {
	return ctrl.GetConfig()
}

var _ caCertGetter = &CACertGetter{}

// CACertGetter can get the CA data used locally to talk to a Kubernetes API.
type CACertGetter struct{}

// GetCACert gets the CAData from a config or reads off the file CAFile.
func (g *CACertGetter) GetCACert(cfg *rest.Config) ([]byte, error) {
	var caData []byte
	var err error

	if len(cfg.CAData) > 0 {
		caData = cfg.CAData
	}
	if cfg.CAFile != "" {
		caData, err = os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, err
		}
	}
	if _, err := certutil.NewPoolFromBytes(caData); err != nil {
		return nil, fmt.Errorf("expected to load root CA config, but got err: %v", err)
	}

	return caData, nil
}
