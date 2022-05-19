// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	tokenKey    = "token"
	caCert      = "ca.crt"
	saTokenType = "kubernetes.io/service-account-token"
)

// ServiceAccountLoader allows the construction of a k8s client from a Service Account
type ServiceAccountLoader struct {
	client client.Client // Used to load service accounts and their secrets.
}

// NewServiceAccountLoader creates a new ServiceAccountLoader
func NewServiceAccountLoader(client client.Client) *ServiceAccountLoader {
	return &ServiceAccountLoader{client}
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

	token, cert, err := s.serviceAccountCredentials(ctx, saName, saNamespace)
	if err != nil {
		return nil, err
	}

	if _, err := certutil.NewPoolFromBytes(cert); err != nil {
		return nil, fmt.Errorf("expected to load root CA config, but got err: %v", err)
	}

	return &rest.Config{
		Host: cfg.Host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: cert,
		},
		BearerToken: string(token),
	}, nil
}

func (s *ServiceAccountLoader) serviceAccountCredentials(ctx context.Context, name, namespace string) ([]byte, []byte, error) {
	sa := corev1.ServiceAccount{}
	if err := s.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &sa); err != nil {
		return nil, nil, fmt.Errorf("unable to fetch service account %s:%s, %w", namespace, name, err)
	}

	if len(sa.Secrets) == 0 {
		return nil, nil, fmt.Errorf("no secrets found for service account %s:%s", namespace, name)
	}

	//TODO what to do if there are mutiple secrets?
	secretName := sa.Secrets[0].Name

	secret := corev1.Secret{}
	if err := s.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, &secret); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch secret %s:%s, %w", namespace, secretName, err)
	}

	if secret.Type != saTokenType {
		return nil, nil, fmt.Errorf("secret %s:%s is not of type %s", namespace, secretName, saTokenType)
	}

	tokenData, tokenFound := secret.Data[tokenKey]
	if !tokenFound {
		return nil, nil, fmt.Errorf("secret %s:%s does not contain %s field", namespace, secretName, tokenKey)
	}

	certData, certFound := secret.Data[caCert]
	if !certFound {
		return nil, nil, fmt.Errorf("secret %s:%s does not contain %s field", namespace, secretName, caCert)
	}

	return tokenData, certData, nil
}
