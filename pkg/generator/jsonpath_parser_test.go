// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"reflect"

	"testing"

	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/generator"
)

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
		{expression: "$($(foo))", expected: "{$(foo)}"},
		{expression: "$(.data.value)-middle-$(.data.value2)", expected: "{.data.value}-middle-{.data.value2}"},
		{
			expression: "$(.pod.spec.containers[?(@.name=='first-filter')].env[?(@.name=='second-filter')].valueFrom.secretKeyRef.name)",
			expected:   "{.pod.spec.containers[?(@.name=='first-filter')].env[?(@.name=='second-filter')].valueFrom.secretKeyRef.name}",
		},
		{expression: "$(.data.foo)-)", expected: "{.data.foo}-)"},
	}

	for _, tc := range tests {
		expression := generator.JSONPath(tc.expression)
		result := expression.ToK8sJSONPath()
		if !reflect.DeepEqual(result, tc.expected) {
			t.Fatalf("expected: %v, got: %v", tc.expected, result)
		}
	}
}

func Test_SecretTemplate_CountDelimiterPairs(t *testing.T) {
	type test struct {
		expression string
		count      int
	}

	tests := []test{
		{expression: "static-value", count: 0},
		{expression: "$(.value)", count: 1},
		{expression: "prefix-$(.value)-suffix", count: 1},
		{expression: "$(.spec.ports[?(@.protocol=='TCP')])", count: 1},

		{expression: "$foo", count: 0},
		{expression: "foo$(", count: 0}, //error?
		{expression: "foo)", count: 0},  // ?
		{expression: "$($(foo))", count: 1},

		//failing
		{expression: "$(.data.value)-middle-$(.data.value2)", count: 2},
		{expression: "$(.data.foo)-)", count: 1},
	}

	for _, tc := range tests {
		expression := generator.JSONPath(tc.expression)
		result := expression.CountDelimiterPairs()
		if !reflect.DeepEqual(result, tc.count) {
			t.Fatalf("expression: %s, expected: %d, got: %d", tc.expression, tc.count, result)
		}
	}
}
