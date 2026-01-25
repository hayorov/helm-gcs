// Copyright © 2024 helm-gcs contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestGetterBinaryArgs(t *testing.T) {
	binary := os.Getenv("HELM_GCS_GETTER_BINARY")
	if binary == "" {
		t.Skip("HELM_GCS_GETTER_BINARY not set, skipping integration test")
	}

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectInOutput string
	}{
		{
			name:           "no args shows usage",
			args:           []string{},
			expectError:    true,
			expectInOutput: "Usage:",
		},
		{
			name:           "version flag",
			args:           []string{"--version"},
			expectError:    false,
			expectInOutput: "helm-gcs-getter",
		},
		{
			name:           "too few args",
			args:           []string{"cert", "key", "ca"},
			expectError:    true,
			expectInOutput: "Usage:",
		},
		{
			name:           "invalid protocol",
			args:           []string{"", "", "", "https://example.com/chart.tgz"},
			expectError:    true,
			expectInOutput: "unsupported protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binary, tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !strings.Contains(string(output), tt.expectInOutput) {
				t.Errorf("expected output to contain %q, got: %s", tt.expectInOutput, output)
			}
		})
	}
}
