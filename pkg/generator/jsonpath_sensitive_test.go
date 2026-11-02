// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"carvel.dev/secretgen-controller/pkg/generator"
)

func TestJSONPathErrorsDoNotExposeInputValues(t *testing.T) {
	secret := "synthetic-password-canary"
	encoded := base64.StdEncoding.EncodeToString([]byte(secret))
	for _, value := range []interface{}{map[string]interface{}{"password": encoded}, secret, encoded} {
		_, err := generator.JSONPath("$(.input[?(@.password=='x')])").EvaluateWith(map[string]interface{}{"input": value})
		if err == nil {
			t.Fatal("expected invalid filter to fail")
		}
		for _, sensitive := range []string{secret, encoded} {
			if strings.Contains(err.Error(), sensitive) {
				t.Fatal("evaluation error disclosed input")
			}
		}
		if !strings.Contains(err.Error(), "filters are applied to arrays") {
			t.Fatal("missing corrective step")
		}
	}
	// The diagnostic change must not redact successful template output.
	result, err := generator.JSONPath("$(.password)").EvaluateWith(map[string]interface{}{"password": secret})
	if err != nil || result.String() != secret {
		t.Fatalf("successful evaluation changed: %v", err)
	}
}
