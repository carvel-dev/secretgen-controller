// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/generator"
)

func Test_SecretTemplate_EvaluateWith(t *testing.T) {
	type test struct {
		expression string
		values     map[string]interface{}
		expected   string
	}

	// Shouldn't test too much here as it's a passing through to a k8s library.
	tests := []test{
		{expression: "static-value", values: map[string]interface{}{
			"key": "value",
		}, expected: "static-value"},
		{expression: "$(.key)", values: map[string]interface{}{
			"key": "value",
		}, expected: "value"},
		{expression: "$(.key)chain", values: map[string]interface{}{
			"key": "value",
		}, expected: "valuechain"},
	}

	for _, tc := range tests {
		expression := generator.JSONPath(tc.expression)
		result, err := expression.EvaluateWith(tc.values)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, result.String())
	}
}

func Test_SecretTemplate_Templating_Syntax(t *testing.T) {
	type test struct {
		expression string
		expected   string
	}

	tests := []test{
		{expression: "static-value", expected: "static-value"},
		{expression: "$(.value)", expected: "{.value}"},
		{expression: "prefix-$(.value)-suffix", expected: "prefix-{.value}-suffix"},
		{expression: "$(.spec.ports[?(@.protocol=='TCP')])", expected: "{.spec.ports[?(@.protocol=='TCP')]}"},
		{expression: "$foo", expected: "$foo"},
		{expression: "foo$(", expected: "foo$("},
		{expression: "foo)", expected: "foo)"},
		{expression: "$($(foo))", expected: "{{foo}}"},
		{expression: "$(.data.value)-middle-$(.data.value2)", expected: "{.data.value}-middle-{.data.value2}"},
		{
			expression: "$(.pod.spec.containers[?(@.name=='first-filter')].env[?(@.name=='second-filter')].valueFrom.secretKeyRef.name)",
			expected:   "{.pod.spec.containers[?(@.name=='first-filter')].env[?(@.name=='second-filter')].valueFrom.secretKeyRef.name}",
		},
		{expression: "$(.data.foo)-)", expected: "{.data.foo}-)"},
		{expression: "$(.data.foo?())()-)", expected: "{.data.foo?()}()-)"},
		{expression: "{.data.foo}", expected: "{.data.foo}"},
		{expression: "$(.items[(@.length-1)])", expected: "{.items[(@.length-1)]}"},
	}

	for _, tc := range tests {
		expression := generator.JSONPath(tc.expression)
		result := expression.ToK8sJSONPath()
		assert.Equal(t, tc.expected, result)
	}
}
