/*
 * Copyright 2025 Adrien Kara
 *
 * This file is part of GoULC.
 *
 * This is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

package client

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// discardLogger keeps the expected close warnings out of the test output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestClose_StopsWaitPoller pins the active request counter above zero so
// the close wait can only end by timeout. The polling goroutine must not
// outlive Close, it used to spin forever without a stop signal.
func TestClose_StopsWaitPoller(t *testing.T) {
	defer goleak.VerifyNone(t)

	opt := OptDefault
	opt.Timeout = 50 * time.Millisecond

	c, err := New(context.Background(), "https://vault.example",
		nil, &opt, discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	c.activeRequests.Store(1)

	if err := c.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestDo_DoesNotRegisterPerRequestCloser guards against the per-request
// closer retention: request snapshots must not be registered on the
// parent, only explicit clones are.
func TestDo_DoesNotRegisterPerRequestCloser(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		}))
	defer ts.Close()

	opt := OptDefault
	opt.DisableTLSVerify = true

	c, err := New(context.Background(), ts.URL, nil, &opt, discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const requests = 5
	for range requests {
		if _, err := c.Do(http.MethodGet, nil, nil); err != nil {
			t.Fatalf("Do() error = %v", err)
		}
	}

	c.mu.RLock()
	got := len(c.closer)
	c.mu.RUnlock()
	if got != 0 {
		t.Errorf("closer count after %d Do() = %d, want 0", requests, got)
	}

	// Explicit clones must still be registered for cascade closing
	if clone := c.Clone(); clone == nil {
		t.Fatal("Clone() = nil, want a clone")
	}

	c.mu.RLock()
	got = len(c.closer)
	c.mu.RUnlock()
	if got != 1 {
		t.Errorf("closer count after Clone() = %d, want 1", got)
	}

	if err := c.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
