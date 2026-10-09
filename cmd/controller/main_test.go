// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

func Test_normalizeNamespace(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"secretgen-controller", "secretgen-controller", false},
		{"SecretGen-Ctrl", "", true},
		{"  My-Ns \n", "", true},
		{"  valid-namespace  ", "valid-namespace", false},
	}

	for _, tt := range tests {
		got, err := normalizeNamespace(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("normalizeNamespace(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("normalizeNamespace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
