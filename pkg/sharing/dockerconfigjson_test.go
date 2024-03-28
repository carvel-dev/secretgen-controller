// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NewCombinedDockerConfigJSON(t *testing.T) {
	t.Run("returns error when secret does not contain parseable auth section", func(t *testing.T) {
		_, err := NewCombinedDockerConfigJSON([]*corev1.Secret{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret1",
				Namespace: "ns1",
			},
		}})
		assert.Error(t, err)
		assert.EqualError(t, err, "Unmarshaling secret 'ns1/secret1': unexpected end of JSON input")
	})

	t.Run("returns combined set of credentials, preferring last secret for duplicate servers", func(t *testing.T) {
		secrets := []*corev1.Secret{
			// 3rd secret overrides 'server' auth creds
			&corev1.Secret{
				Data: map[string][]byte{
					"George":                   []byte("Washington"),
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"server":{"username":"TopSecret","password":"password1","auth":"author"}}}`),
				},
			},
			// Secret without auths
			&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{}`),
				},
			},
			// Secret with 0 auths
			&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{}}`),
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"server2":{"username":"user2","password":"password2","auth":"auth2"}}}`),
				},
			},
			&corev1.Secret{
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"server":{"username":"correctUser","password":"correctPassword","auth":"correctAuth"}}}`),
				},
			},
		}
		result, err := NewCombinedDockerConfigJSON(secrets)
		require.NoError(t, err)

		expected := []byte(`{"auths":{"server":{"username":"correctUser","password":"correctPassword","auth":"correctAuth"},"server2":{"username":"user2","password":"password2","auth":"auth2"}}}`)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, expected, result[corev1.DockerConfigJsonKey])
	})
}
