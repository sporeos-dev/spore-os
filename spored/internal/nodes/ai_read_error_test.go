// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"errors"
	"testing"
)

type testReadError struct {
	timeout   bool
	temporary bool
}

func (e testReadError) Error() string   { return "test read error" }
func (e testReadError) Timeout() bool   { return e.timeout }
func (e testReadError) Temporary() bool { return e.temporary }

func TestIsRetryableReadError(t *testing.T) {
	tests := []struct {
		name      string
		err       readError
		retryable bool
	}{
		{name: "timeout", err: testReadError{timeout: true}, retryable: true},
		{name: "timeout and temporary", err: testReadError{timeout: true, temporary: true}, retryable: true},
		{name: "temporary only", err: testReadError{temporary: true}},
		{name: "other network error", err: testReadError{}},
		{name: "non-network error", err: errors.New("read error")},
		{name: "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableReadError(tt.err); got != tt.retryable {
				t.Fatalf("isRetryableReadError() = %t, want %t", got, tt.retryable)
			}
		})
	}
}
