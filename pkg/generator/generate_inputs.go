// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"encoding/json"
	"errors"
)

const (
	GenerateInputsAnnKey = "secretgen.k14s.io/generate-inputs"
)

type GenerateInputs struct {
	inputs interface{}
}

func (i GenerateInputs) Add(anns map[string]string) error {
	if anns == nil {
		return errors.New("internal inconsistency: called with annotations nil param")
	}
	bs, err := json.Marshal(i.inputs)
	if err != nil {
		return errors.New("cannot marshal generate inputs")
	}
	anns[GenerateInputsAnnKey] = string(bs)
	return nil
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
