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
	"strings"
	"testing"
)

func TestVersionStringFormat(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		commit   string
		date     string
		expected string
	}{
		{
			name:     "dev version with empty commit and date",
			version:  "dev",
			commit:   "",
			date:     "",
			expected: "helm-gcs-getter dev (commit: , date: )",
		},
		{
			name:     "release version with commit and date",
			version:  "1.0.0",
			commit:   "abc1234",
			date:     "2024-01-15",
			expected: "helm-gcs-getter 1.0.0 (commit: abc1234, date: 2024-01-15)",
		},
		{
			name:     "version with only commit",
			version:  "0.7.0",
			commit:   "deadbeef",
			date:     "",
			expected: "helm-gcs-getter 0.7.0 (commit: deadbeef, date: )",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatVersion(tt.version, tt.commit, tt.date)
			if result != tt.expected {
				t.Errorf("formatVersion() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func formatVersion(ver, commit, date string) string {
	return "helm-gcs-getter " + ver + " (commit: " + commit + ", date: " + date + ")"
}

func TestURLProtocolValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		isValid bool
	}{
		{
			name:    "valid gs:// protocol",
			url:     "gs://my-bucket/charts/mychart-0.1.0.tgz",
			isValid: true,
		},
		{
			name:    "valid gcs:// protocol",
			url:     "gcs://my-bucket/charts/mychart-0.1.0.tgz",
			isValid: true,
		},
		{
			name:    "valid gs:// with minimal path",
			url:     "gs://bucket/index.yaml",
			isValid: true,
		},
		{
			name:    "valid gcs:// with deep path",
			url:     "gcs://bucket/a/b/c/d/chart.tgz",
			isValid: true,
		},
		{
			name:    "invalid http:// protocol",
			url:     "http://example.com/charts/mychart-0.1.0.tgz",
			isValid: false,
		},
		{
			name:    "invalid https:// protocol",
			url:     "https://storage.googleapis.com/bucket/chart.tgz",
			isValid: false,
		},
		{
			name:    "invalid s3:// protocol",
			url:     "s3://my-bucket/charts/mychart-0.1.0.tgz",
			isValid: false,
		},
		{
			name:    "invalid empty URL",
			url:     "",
			isValid: false,
		},
		{
			name:    "invalid file:// protocol",
			url:     "file:///path/to/chart.tgz",
			isValid: false,
		},
		{
			name:    "invalid plain path",
			url:     "/path/to/chart.tgz",
			isValid: false,
		},
		{
			name:    "case sensitive - GS:// is invalid",
			url:     "GS://bucket/chart.tgz",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidGCSProtocol(tt.url)
			if result != tt.isValid {
				t.Errorf("isValidGCSProtocol(%q) = %v, want %v", tt.url, result, tt.isValid)
			}
		})
	}
}

func isValidGCSProtocol(url string) bool {
	return strings.HasPrefix(url, "gs://") || strings.HasPrefix(url, "gcs://")
}

func TestArgumentValidation(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		errorType string
	}{
		{
			name:      "valid arguments with 5 args",
			args:      []string{"helm-gcs-getter", "cert", "key", "ca", "gs://bucket/chart.tgz"},
			wantError: false,
		},
		{
			name:      "too few arguments - 4 args",
			args:      []string{"helm-gcs-getter", "cert", "key", "ca"},
			wantError: true,
			errorType: "insufficient_args",
		},
		{
			name:      "too few arguments - 3 args",
			args:      []string{"helm-gcs-getter", "cert", "key"},
			wantError: true,
			errorType: "insufficient_args",
		},
		{
			name:      "too few arguments - 1 arg",
			args:      []string{"helm-gcs-getter"},
			wantError: true,
			errorType: "insufficient_args",
		},
		{
			name:      "version flag only",
			args:      []string{"helm-gcs-getter", "--version"},
			wantError: false,
		},
		{
			name:      "valid args with gcs:// protocol",
			args:      []string{"helm-gcs-getter", "", "", "", "gcs://bucket/path/chart.tgz"},
			wantError: false,
		},
		{
			name:      "invalid protocol in URL",
			args:      []string{"helm-gcs-getter", "cert", "key", "ca", "http://example.com/chart.tgz"},
			wantError: true,
			errorType: "invalid_protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArgs(tt.args)
			if tt.wantError && err == nil {
				t.Errorf("validateArgs() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("validateArgs() unexpected error: %v", err)
			}
		})
	}
}

type argError struct {
	errorType string
	message   string
}

func (e *argError) Error() string {
	return e.message
}

func validateArgs(args []string) error {
	if len(args) == 2 && args[1] == "--version" {
		return nil
	}

	if len(args) < 5 {
		return &argError{
			errorType: "insufficient_args",
			message:   "insufficient arguments",
		}
	}

	href := args[4]
	if !isValidGCSProtocol(href) {
		return &argError{
			errorType: "invalid_protocol",
			message:   "unsupported protocol",
		}
	}

	return nil
}

func TestDownloadWithInvalidServiceAccount(t *testing.T) {
	err := download("gs://test-bucket/test-file.txt", "/nonexistent/path/to/sa.json")
	if err == nil {
		t.Error("download() with invalid service account should return error")
	}
}

func TestDownloadWithInvalidURL(t *testing.T) {
	err := download("invalid-url", "")
	if err == nil {
		t.Error("download() with invalid URL should return error")
	}
}
