package sharing

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

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

		err := json.Unmarshal(secret.Data[corev1.DockerConfigJsonKey], &auths)
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
