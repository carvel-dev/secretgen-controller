package generator

import (
	"encoding/json"
)

const (
	GenerateInputsAnnKey = "secretgen.k14s.io/generate-inputs"
)

type GenerateInputs struct {
	inputs interface{}
}

func (i GenerateInputs) Add(anns map[string]string) {
	bs, err := json.Marshal(i.inputs)
	if err != nil {
		panic("Cannot marshal generate inputs")
	}
	anns[GenerateInputsAnnKey] = string(bs)
}

func (i GenerateInputs) IsChanged(anns map[string]string) bool {
	bs, err := json.Marshal(i.inputs)
	if err != nil {
		panic("Cannot marshal generate inputs")
	}

	existingVal, found := anns[GenerateInputsAnnKey]
	if !found {
		return true
	}

	return string(bs) != existingVal
}
