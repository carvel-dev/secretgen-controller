package types

import (
	"github.com/cloudfoundry/bosh-utils/errors"
	"gopkg.in/yaml.v2"
)

func stringInArray(key string, list []string) bool {
	for _, value := range list {
		if key == value {
			return true
		}
	}
	return false
}

func objToStruct(input interface{}, str interface{}, supportedParameters []string) error {
	valBytes, err := yaml.Marshal(input)
	if err != nil {
		return errors.WrapErrorf(err, "Expected input to be serializable")
	}

	parametersMap := make(map[string]interface{})
	err = yaml.Unmarshal(valBytes, parametersMap)
	if err != nil {
		return errors.WrapErrorf(err, "Expected input to be deserializable")
	}

	for key := range parametersMap {
		if !stringInArray(key, supportedParameters) {
			return errors.Errorf("Unsupported parameter '%s'", key)
		}
	}

	err = yaml.Unmarshal(valBytes, str)
	if err != nil {
		return errors.WrapErrorf(err, "Expected input to be deserializable")
	}

	return nil
}
