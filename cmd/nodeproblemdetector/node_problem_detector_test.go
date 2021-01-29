/*
Copyright 2021 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"strings"
	"testing"

	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/exporterplugins"
	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/problemdaemonplugins"
)

func TestGetProgramPath(t *testing.T) {
	path, err := getProgramPath()

	if err != nil {
		t.Error(err)
	}

	if path == "" {
		t.Error("Path is empty")
	}
}

func TestFilterArgsForService(t *testing.T) {
	var testCases = []struct {
		input    []string
		expected []string
	}{
		{
			input:    []string{},
			expected: []string{},
		},
		{
			input:    []string{"app.exe"},
			expected: []string{},
		},
		{
			input:    []string{"app.exe", "-service-action"},
			expected: []string{},
		},
		{
			input:    []string{"app.exe", "a", "-service-action"},
			expected: []string{"a"},
		},
		{
			input:    []string{"app.exe", "a", "-service-action", "install"},
			expected: []string{"a"},
		},
		{
			input:    []string{"app.exe", "a", "-service-action", "install", "b"},
			expected: []string{"a", "b"},
		},
		{
			input:    []string{"app.exe", "a", "-service-action=install", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			input:    []string{"app.exe", "a", "-service-action=start", "b", "c", "-service-action=start", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
	}
	for _, tt := range testCases {
		tc := tt
		t.Run(fmt.Sprintf("%v", tc.input), func(t *testing.T) {
			t.Parallel()

			actual := filterArgsForService(tc.input)
			args := strings.Join(actual, " ")
			expectedArgs := strings.Join(tc.expected, " ")

			if args != expectedArgs {
				t.Errorf("got %q, want %q", args, expectedArgs)
			}
		})
	}

	testsInTest := getArgsForService()
	if len(testsInTest) == 0 {
		t.Error("args in test were empty")
	}
}
