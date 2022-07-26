// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/generator"
)

func TestAddSucceedsWithEmptyAnnotations(t *testing.T) {
	err := generator.GenerateInputs{}.Add(nil)
	assert.Equal(t, nil, err)
}

func TestAddSuccessfulWithDefaultAnnotation(t *testing.T) {
	defaultAnnotations := map[string]string{
		"secretgen.k14s.io/generate-inputs": "",
	}
	err := generator.GenerateInputs{}.Add(defaultAnnotations)
	assert.Equal(t, nil, err)
}
