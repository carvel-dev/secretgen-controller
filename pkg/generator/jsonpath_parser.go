// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"fmt"
	"strings"
)

const (
	leftDelimiter  = "$("
	rightDelimiter = ")"
)

type JSONPath string

// Count the number of delimiter pairs in the path.
func (p JSONPath) CountDelimiterPairs() int {
	count := 0

	var delimiters stack
	path := string(p)

	for i := range path {
		// If left delimiter and previous was not a left delimiter, then add to stack.
		if i < len(path)-2 && path[i:i+2] == leftDelimiter {
			if delimiters.peek() != leftDelimiter {
				delimiters = delimiters.push(leftDelimiter)
			}
		}

		// If right delimiter and previous was a left delimiter, then pop and count as pair.
		if string(path[i]) == rightDelimiter {
			if delimiters.peek() == leftDelimiter {
				delimiters = delimiters.pop()
				count += 1
			}
		}
	}

	return count
}

// If the expression contains an opening $( and a closing ), toK8sJSONPath will replace them with a { and a } respectively.
func (p JSONPath) ToK8sJSONPath() string {
	newPath := string(p)
	i := 0
	for pair := 0; pair < p.CountDelimiterPairs(); pair++ {
		for string(newPath[i:i+2]) != leftDelimiter {
			i += 1
		}

		if newPath[i:i+2] == leftDelimiter {
			newPath = replace(newPath, i, leftDelimiter, "{")

			// Skip inner filters and inner $() expressions.
			for string(newPath[i]) != rightDelimiter {
				nextTwo := string(newPath[i : i+2])
				if nextTwo == "?(" || nextTwo == leftDelimiter {
					for string(newPath[i]) != rightDelimiter {
						i += 1
					}
				}

				i += 1
			}

			newPath = replace(newPath, i, rightDelimiter, "}")
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

// Helper stack for counting pairs.
type stack []string

func (s stack) push(x string) stack {
	return append(s, x)
}

func (s stack) pop() stack {
	return s[:len(s)-1]
}

func (s stack) peek() string {
	if len(s) == 0 {
		return ""
	}

	return s[len(s)-1]
}
