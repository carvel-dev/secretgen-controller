// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"testing"

	sgv1alpha1 "carvel.dev/secretgen-controller/pkg/apis/secretgen/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordReconciler_Generate(t *testing.T) {
	r := &PasswordReconciler{}

	t.Run("generates password of requested length", func(t *testing.T) {
		pw, err := r.generate(&sgv1alpha1.Password{Spec: sgv1alpha1.PasswordSpec{
			Length: 40, Digits: 5, UppercaseLetters: 5, LowercaseLetters: 5, Symbols: 5, SymbolCharSet: "!@#",
		}})
		require.NoError(t, err)
		assert.Len(t, pw, 40)
	})

	t.Run("accepts the maximum allowed value", func(t *testing.T) {
		pw, err := r.generate(&sgv1alpha1.Password{Spec: sgv1alpha1.PasswordSpec{Length: maxPasswordSpecValue}})
		require.NoError(t, err)
		assert.Len(t, pw, maxPasswordSpecValue)
	})

	t.Run("rejects values above the maximum", func(t *testing.T) {
		over := maxPasswordSpecValue + 1
		specs := map[string]sgv1alpha1.PasswordSpec{
			"length":           {Length: over},
			"huge length":      {Length: 150000000},
			"digits":           {Length: 40, Digits: over},
			"symbols":          {Length: 40, Symbols: over, SymbolCharSet: "!"},
			"uppercaseLetters": {Length: 40, UppercaseLetters: over},
			"lowercaseLetters": {Length: 40, LowercaseLetters: over},
		}
		for name, spec := range specs {
			t.Run(name, func(t *testing.T) {
				pw, err := r.generate(&sgv1alpha1.Password{Spec: spec})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "must each be between 0 and 4096")
				assert.Empty(t, pw)
			})
		}
	})

	t.Run("rejects negative values", func(t *testing.T) {
		_, err := r.generate(&sgv1alpha1.Password{Spec: sgv1alpha1.PasswordSpec{Length: 40, Digits: -1}})
		require.Error(t, err)
	})
}
