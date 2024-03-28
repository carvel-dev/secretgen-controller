// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"
	"strings"
)

func combinedErrs(title string, errs []error) error {
	if len(errs) > 0 {
		var msgs []string
		for _, err := range errs {
			msgs = append(msgs, "- "+err.Error())
		}
		return fmt.Errorf("%s:\n%s", title, strings.Join(msgs, "\n"))
	}
	return nil
}
