// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"errors"
	"testing"

	"carvel.dev/secretgen-controller/pkg/generator"
	"github.com/stretchr/testify/assert"
)

func TestAddFailsWithEmptyAnnotations(t *testing.T) {
	err := generator.GenerateInputs{}.Add(nil)
	assert.Equal(t, errors.New("internal inconsistency: called with annotations nil param"), err)
}

func TestAddSucceedsfulWithDefaultAnnotation(t *testing.T) {
	defaultAnnotations := map[string]string{
		"secretgen.k14s.io/generate-inputs": "",
	}
	err := generator.GenerateInputs{}.Add(defaultAnnotations)
	assert.Equal(t, nil, err)
}
