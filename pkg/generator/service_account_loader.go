// Copyright 2024 The Carvel Authors.
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

// ServiceAccountLoader allows the construction of a k8s client from a Service Account
type ServiceAccountLoader struct {
	// Ensures a valid token for a ServiceAccount is available.
	tokenManager TokenManager
}

// NewServiceAccountLoader creates a new ServiceAccountLoader
func NewServiceAccountLoader(manager TokenManager) *ServiceAccountLoader {
	return &ServiceAccountLoader{manager}
}

// Client returns a new k8s client for a Service Account
func (s *ServiceAccountLoader) Client(ctx context.Context, saName, saNamespace string) (client.Client, error) {
	config, err := s.restConfig(ctx, saName, saNamespace)
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{})
}

func (s *ServiceAccountLoader) restConfig(ctx context.Context, saName, saNamespace string) (*rest.Config, error) {
	cfg, err := ctrl.GetConfig()
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

	caData, err := getCACert(cfg)
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

func getCACert(cfg *rest.Config) ([]byte, error) {
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
