package sharing

import (
	"encoding/json"
	"fmt"
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
	// (TODO?) this is a very verbose test -- i tried just making a byte array of encoded json but then the equality comparison cares about whitespace in a really finicky way...
	type authConf struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Auth     string `json:"auth"`
	}

	type authsConf struct {
		Auths map[string]authConf `json:"auths"`
	}

	auths := authsConf{
		Auths: map[string]authConf{
			"server": authConf{
				Username: "TopSecret",
				Password: "password1",
				Auth:     "author",
			},
		},
	}

	jsonAuth, err := json.Marshal(auths)
	require.NoError(t, err)

	secrets := []*corev1.Secret{
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			},
			Data: map[string][]byte{
				"George":                   []byte("Washington"),
				corev1.DockerConfigJsonKey: jsonAuth,
			},
		},
	}

	result, err := NewCombinedDockerConfigJSON(secrets)

	require.NoError(t, err)
	fmt.Println("result: ", result)
	fmt.Println("result[.dockerconfig]: ", string(result[".dockerconfigjson"]))
	assert.Equal(t, 1, len(result))
	var observedAuths authsConf
	err = json.Unmarshal(result[".dockerconfigjson"], &observedAuths)
	require.NoError(t, err)
	assert.Equal(t, auths, observedAuths)
}
