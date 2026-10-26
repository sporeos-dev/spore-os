// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import "testing"

type testReadError struct {
	timeout   bool
	temporary bool
}

func (e testReadError) Error() string   { return "test read error" }
func (e testReadError) Timeout() bool   { return e.timeout }
func (e testReadError) Temporary() bool { return e.temporary }

func TestIsRetryableReadError(t *testing.T) {
	if !isRetryableReadError(testReadError{timeout: true}) {
		t.Fatal("timeout read error should be retryable")
	}
	if !isRetryableReadError(testReadError{temporary: true}) {
		t.Fatal("temporary read error should be retryable")
	}
	if isRetryableReadError(testReadError{}) {
		t.Fatal("terminal read error should not be retryable")
	}
}
