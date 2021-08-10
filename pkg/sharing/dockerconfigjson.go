package sharing

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// NewCombinedDockerConfigJSON combines multiple kubernetes.io/dockerconfigjson Secrets
// into a single map to be used in single Secret.
// (https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets)
func NewCombinedDockerConfigJSON(secrets []*corev1.Secret) (map[string][]byte, error) {
	type authConf struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Auth     string `json:"auth"`
	}

	type authsConf struct {
		Auths map[string]authConf `json:"auths"`
	}

	combined := authsConf{
		Auths: map[string]authConf{},
	}

	// TODO deterministically order secrets
	for _, secret := range secrets {
		var auths authsConf

		secretData := secret.Data[corev1.DockerConfigJsonKey]
		if len(secretData) == 0 {
			// TODO: if we actually like this behavior this should be a warning through the logger
			fmt.Println("Skipping Empty Secret")
			continue
		}
		err := json.Unmarshal(secretData, &auths)
		if err != nil {
			return nil, fmt.Errorf("Unmarshaling secret '%s/%s': %s", secret.Namespace, secret.Name, err)
		}

		// TODO should we have more complex merging here?
		for server, auth := range auths.Auths {
			combined.Auths[server] = auth
		}
	}

	encodedCombined, err := json.Marshal(combined)
	if err != nil {
		return nil, fmt.Errorf("Marshaling combined secret: %s", err)
	}

	return map[string][]byte{corev1.DockerConfigJsonKey: encodedCombined}, nil
}
