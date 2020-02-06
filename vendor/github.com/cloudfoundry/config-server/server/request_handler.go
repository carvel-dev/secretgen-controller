package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/config-server/store"
	"github.com/cloudfoundry/config-server/types"
	"net/http"
	"regexp"
	"strings"
)

type requestHandler struct {
	store                 store.Store
	valueGeneratorFactory types.ValueGeneratorFactory
}

func NewRequestHandler(store store.Store, valueGeneratorFactory types.ValueGeneratorFactory) (http.Handler, error) {
	if store == nil {
		return nil, errors.Error("Data store must be set")
	}
	return requestHandler{
		store:                 store,
		valueGeneratorFactory: valueGeneratorFactory,
	}, nil
}

func (handler requestHandler) ServeHTTP(resWriter http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		handler.handleGet(resWriter, req)
	case "PUT":
		handler.handlePut(resWriter, req)
	case "POST":
		handler.handlePost(resWriter, req)
	case "DELETE":
		handler.handleDelete(resWriter, req)
	default:
		http.Error(resWriter, NewErrorResponse(errors.Error("HTTP method not allowed")).GenerateErrorMsg(), http.StatusMethodNotAllowed)
	}
}

func (handler requestHandler) handleGet(resWriter http.ResponseWriter, req *http.Request) {
	id, idErr := extractIDFromURLPath(req.URL.Path)
	if idErr == nil {
		handler.handleGetByID(id, resWriter)
	} else {
		name := req.URL.Query().Get("name")
		if len(name) == 0 {
			http.Error(resWriter, NewErrorResponse(idErr).GenerateErrorMsg(), http.StatusBadRequest)
		} else {
			handler.handleGetByName(name, resWriter)
		}
	}
}

func (handler requestHandler) handleGetByID(id string, resWriter http.ResponseWriter) {

	value, err := handler.store.GetByID(id)

	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusInternalServerError)
		return
	}

	emptyValue := store.Configuration{}

	if value == emptyValue {
		http.Error(resWriter, NewErrorResponse(errors.Errorf("ID '%s' not found", id)).GenerateErrorMsg(), http.StatusNotFound)
	} else {
		result, _ := value.StringifiedJSON()
		respond(resWriter, result, http.StatusOK)
	}
}

func (handler requestHandler) handleGetByName(name string, resWriter http.ResponseWriter) {

	if isNameValid, nameError := isValidName(name); isNameValid == false {
		http.Error(resWriter, NewErrorResponse(nameError).GenerateErrorMsg(), http.StatusBadRequest)
		return
	}

	values, err := handler.store.GetByName(name)
	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusInternalServerError)
		return
	}

	if len(values) == 0 {
		http.Error(resWriter, NewErrorResponse(errors.Errorf("Name '%s' not found", name)).GenerateErrorMsg(), http.StatusNotFound)
	} else {
		result, err := store.Configurations(values).StringifiedJSON()
		if err == nil {
			respond(resWriter, result, http.StatusOK)
		}
	}
}

func (handler requestHandler) handlePut(resWriter http.ResponseWriter, req *http.Request) {
	if contentTypeErr := validateRequestContentType(req); contentTypeErr != nil {
		http.Error(resWriter, NewErrorResponse(contentTypeErr).GenerateErrorMsg(), http.StatusUnsupportedMediaType)
		return
	}

	name, value, err := readPutRequest(req)

	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusBadRequest)
		return
	}

	configuration, err := handler.saveToStore(name, value, "")

	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusInternalServerError)
		return
	}
	result, _ := configuration.StringifiedJSON()
	respond(resWriter, result, http.StatusOK)
}

func (handler requestHandler) handlePost(resWriter http.ResponseWriter, req *http.Request) {
	if contentTypeErr := validateRequestContentType(req); contentTypeErr != nil {
		http.Error(resWriter, NewErrorResponse(contentTypeErr).GenerateErrorMsg(), http.StatusUnsupportedMediaType)
		return
	}

	name, generatorType, parameters, mode, err := readPostRequest(req)

	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusBadRequest)
		return
	}

	values, err := handler.store.GetByName(name)
	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusInternalServerError)
		return
	}

	checksum, err := handler.calculateChecksum(parameters)
	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusInternalServerError)
	}

	if len(values) != 0 {
		configuration := values[0]
		if "converge" != mode || checksum == configuration.ParameterChecksum {
			result, err := configuration.StringifiedJSON()
			if err != nil {
				http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusInternalServerError)
			} else {
				respond(resWriter, result, http.StatusOK)
			}
			return
		}
	}

	generator, err := handler.valueGeneratorFactory.GetGenerator(generatorType)
	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusBadRequest)
		return
	}

	generatedValue, err := generator.Generate(parameters)
	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusBadRequest)
		return
	}

	configuration, err := handler.saveToStore(name, generatedValue, checksum)
	if err != nil {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusInternalServerError)
		return
	}

	result, _ := configuration.StringifiedJSON()
	respond(resWriter, result, http.StatusCreated)
}

func (handler requestHandler) calculateChecksum(v interface{}) (string, error) {
	result, err := json.Marshal(v)
	if err != nil {
		return "", errors.WrapError(err, "Calculating checksum:")
	}

	checksum := sha256.New().Sum(result)
	return hex.EncodeToString(checksum), nil
}

func (handler requestHandler) handleDelete(resWriter http.ResponseWriter, req *http.Request) {
	name := req.URL.Query().Get("name")
	if isNameValid, nameError := isValidName(name); isNameValid == false {
		http.Error(resWriter, NewErrorResponse(nameError).GenerateErrorMsg(), http.StatusBadRequest)
		return
	}

	deleted, err := handler.store.Delete(name)

	if err == nil {
		if deleted == 0 {
			http.Error(resWriter, NewErrorResponse(errors.Errorf("Name '%s' not found", name)).GenerateErrorMsg(), http.StatusNotFound)
		} else {
			respond(resWriter, "", http.StatusNoContent)
		}
	} else {
		http.Error(resWriter, NewErrorResponse(err).GenerateErrorMsg(), http.StatusInternalServerError)
	}
}

func (handler requestHandler) saveToStore(name string, value interface{}, checksum string) (store.Configuration, error) {
	configValue := make(map[string]interface{})
	configValue["value"] = value

	bytes, err := json.Marshal(&configValue)

	if err != nil {
		return store.Configuration{}, err
	}

	id, err := handler.store.Put(name, string(bytes), checksum)
	if err != nil {
		return store.Configuration{}, err
	}

	configuration, err := handler.store.GetByID(id)
	return configuration, err
}

func respond(res http.ResponseWriter, message string, status int) {
	res.WriteHeader(status)

	_, err := res.Write([]byte(message))
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
}

func readPutRequest(req *http.Request) (string, interface{}, error) {

	jsonMap, err := readJSONBody(req)
	if err != nil {
		return "", nil, err
	}

	name, err := getStringValueFromJSONBody(jsonMap, "name")
	if err != nil {
		return "", nil, err
	}

	if isNameValid, nameError := isValidName(name); isNameValid == false {
		return "", nil, nameError
	}

	value, keyExists := jsonMap["value"]
	if !keyExists {
		return "", nil, errors.Error("JSON request body should contain the key 'value'")
	}

	return name, value, nil
}

func readPostRequest(req *http.Request) (string, string, interface{}, string, error) {
	jsonMap, err := readJSONBody(req)
	if err != nil {
		return "", "", nil, "", err
	}

	name, err := getStringValueFromJSONBody(jsonMap, "name")
	if err != nil {
		return name, "", nil, "", err
	}

	generatorType, err := getStringValueFromJSONBody(jsonMap, "type")
	if err != nil {
		return name, generatorType, nil, "", err
	}

	mode, err := getOptionalStringValueFromJSONBody(jsonMap, "mode", "no-overwrite")
	if err != nil {
		return name, generatorType, nil, "", err
	}

	return name, generatorType, jsonMap["parameters"], mode, nil
}

func getOptionalStringValueFromJSONBody(jsonMap map[string]interface{}, keyName string, defaultValue string) (string, error) {
	value, keyExists := jsonMap[keyName]
	if !keyExists {
		return defaultValue, nil
	}

	switch value.(type) {
	case string:
		return value.(string), nil
	default:
		return "", errors.Error(fmt.Sprintf("JSON request body key '%s' must be of type string", keyName))
	}
}

func getStringValueFromJSONBody(jsonMap map[string]interface{}, keyName string) (string, error) {

	value, keyExists := jsonMap[keyName]
	if !keyExists {
		return "", errors.Error(fmt.Sprintf("JSON request body should contain the key '%s'", keyName))
	}

	switch value.(type) {
	case string:
		return value.(string), nil
	default:
		return "", errors.Error(fmt.Sprintf("JSON request body key '%s' must be of type string", keyName))
	}
}

func readJSONBody(req *http.Request) (map[string]interface{}, error) {
	if req == nil {
		return nil, errors.Error("Request can't be nil")
	}

	if req.Body == nil {
		return nil, errors.Error("Request can't be empty")
	}

	var f interface{}
	if err := json.NewDecoder(req.Body).Decode(&f); err != nil {
		return nil, errors.Error("Request Body should be JSON string")
	}

	return f.(map[string]interface{}), nil
}

func extractIDFromURLPath(path string) (string, error) {
	paths := strings.Split(strings.Trim(path, "/"), "/")

	if len(paths) < 3 {
		return "", errors.Error("Request URL invalid, seems to be missing ID")
	}

	id := paths[len(paths)-1]
	if len(id) == 0 {
		return "", errors.Error("Request URL invalid, seems to be missing ID")
	}
	return id, nil
}

func isValidName(name string) (bool, error) {
	var validNameToken = regexp.MustCompile(`^[a-zA-Z0-9_\-\/]+$`)
	if !validNameToken.MatchString(name) {
		return false, errors.Error("Name must consist of alphanumeric, underscores, dashes, and forward slashes")
	}

	return true, nil
}

func validateRequestContentType(req *http.Request) error {
	if !strings.EqualFold(req.Header.Get("content-type"), "application/json") {
		return errors.Error("Unsupported Media Type - Accepts application/json only")
	}

	return nil
}
