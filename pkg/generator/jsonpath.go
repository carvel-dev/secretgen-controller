// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"bytes"
	"fmt"
	"strings"

	"k8s.io/client-go/util/jsonpath"
)

const (
	openPrefix    = "$"
	open          = "("
	close         = ")"
	jsonPathOpen  = "{"
	jsonPathClose = "}"
)

type JSONPath string

// EvaluateWith an expression with respect to values and return the result.
func (p JSONPath) EvaluateWith(values interface{}) (*bytes.Buffer, error) {
	parser := jsonpath.New("").AllowMissingKeys(false)
	err := parser.Parse(p.ToK8sJSONPath())
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = parser.Execute(buf, values)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// If the expression contains an opening $( and a closing ), toK8sJSONPathExpression will replace them with a { and a } respectively.
func (p JSONPath) ToK8sJSONPath() string {
	newPath := string(p)
	var openPositions positions

	for i := 0; i < len(newPath); i++ {
		switch string(newPath[i]) {
		case open:
			openPositions = openPositions.push(i)
		case close:
			d := openPositions.peek()
			if d > 0 && string(newPath[d-1]) == openPrefix {
				newPath = replace(newPath, d-1, openPrefix+open, jsonPathOpen)
				i = i - 1 //Removed a character, fix i
				newPath = replace(newPath, i, close, jsonPathClose)
			}

			openPositions = openPositions.pop()
		}
	}
	return newPath
}

// In string s, replace the substr old, at index i, with substr new.
func replace(s string, i int, old, new string) string {
	if i+len(old) > len(s) {
		return fmt.Sprintf("%s}", s[0:i])
	}
	return strings.Join([]string{s[0:i], s[i+len(old):]}, new)
}

type positions []int

func (s positions) push(position int) positions {
	return append(s, position)
}

func (s positions) pop() positions {
	if s.peek() == -1 {
		return s
	} else {
		return s[:len(s)-1]
	}
}

func (s positions) peek() int {
	if len(s) == 0 {
		return -1
	}

	return s[len(s)-1]
}
