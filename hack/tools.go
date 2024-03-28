// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0
//go:build tools
// +build tools

package tools

import (
	_ "k8s.io/code-generator"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
