package sharing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NewCombinedDockerConfigJSON_handlesEmptySecrets(t *testing.T) {
	secrets := []*corev1.Secret{
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			}},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			}},
	}
	result, err := NewCombinedDockerConfigJSON(secrets)
	require.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, `{"auths":{}}`, string(result[".dockerconfigjson"]))
}

func Test_NewCombinedDockerConfigJSON_happyPath(t *testing.T) {
	secrets := []*corev1.Secret{
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			},
			Data: map[string][]byte{
				"George":                   []byte("Washington"), // third secret also has 'server' so we're testing that it overrides this secret's settings
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"server":{"username":"TopSecret","password":"password1","auth":"author"}}}`),
			},
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"server2":{"username":"user2","password":"password2","auth":"auth2"}}}`),
			},
		},
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			},
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: []byte(`{"auths":{"server":{"username":"correctUser","password":"correctPassword","auth":"correctAuth"}}}`),
			},
		},
	}
	result, err := NewCombinedDockerConfigJSON(secrets)
	require.NoError(t, err)
	assert.Equal(t, 1, len(result))
	expected := []byte(`{"auths":{"server":{"username":"correctUser","password":"correctPassword","auth":"correctAuth"},"server2":{"username":"user2","password":"password2","auth":"auth2"}}}`)
	assert.Equal(t, expected, result[".dockerconfigjson"])
}
