/*
 * Copyright 2026 Adrien Kara
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

package oauth2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client"
)

// TestClientCredentials_UpdateExpiryMargin pins the clock to prove the
// refresh decision: a token is kept while it lives beyond the margin,
// refreshed as soon as it enters it, and the new expiry is computed
// from expires_in interpreted as seconds (RFC 6749 §5.1).
func TestClientCredentials_UpdateExpiryMargin(t *testing.T) {
	fixedNow := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		untilExpiry time.Duration
		wantFetch   bool
	}{
		{
			name:        "token valid beyond the margin is kept",
			untilExpiry: expiryMargin + time.Second,
			wantFetch:   false,
		},
		{
			name:        "token within the margin is refreshed",
			untilExpiry: expiryMargin - time.Second,
			wantFetch:   true,
		},
		{
			name:        "expired token is refreshed",
			untilExpiry: -time.Second,
			wantFetch:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					hits.Add(1)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{
						"access_token": "fresh-token",
						"token_type": "Bearer",
						"expires_in": 3600
					}`))
				}))
			defer server.Close()

			httpClient, err := client.New(
				context.Background(), server.URL, nil, &client.Options{
					OnlyHTTPS:        false,
					DisableTLSVerify: true,
					Timeout:          time.Minute,
				}, nil)
			if err != nil {
				t.Fatalf("client.New() unexpected error: %v", err)
			}
			defer func() { _ = httpClient.Close() }()

			cc, err := NewClientCredentials(ClientInHeader, Config{
				ClientID:     "test-client",
				ClientSecret: hided.NewString("test-secret"),
				Endpoint:     Endpoint{URL: server.URL, Auth: "/oauth/token"},
			}, nil, &httpClient)
			if err != nil {
				t.Fatalf("NewClientCredentials() unexpected error: %v", err)
			}

			cc.now = func() time.Time { return fixedNow }
			cc.Token = TokenResponse{
				Token:    hided.NewString("old-token"),
				ExpireAt: fixedNow.Add(tt.untilExpiry),
			}

			if err := cc.Update(); err != nil {
				t.Fatalf("Update() unexpected error: %v", err)
			}

			wantHits := int32(0)
			if tt.wantFetch {
				wantHits = 1
			}
			if got := hits.Load(); got != wantHits {
				t.Errorf("token endpoint hits = %d, want %d", got, wantHits)
			}

			if !tt.wantFetch {
				if got := cc.Token.Token.Reveal(); got != "old-token" {
					t.Errorf("token = %q, want the cached %q", got, "old-token")
				}
				return
			}

			if got := cc.Token.Token.Reveal(); got != "fresh-token" {
				t.Errorf("token = %q, want %q", got, "fresh-token")
			}
			if want := fixedNow.Add(3600 * time.Second); !cc.Token.ExpireAt.Equal(want) {
				t.Errorf("ExpireAt = %v, want %v", cc.Token.ExpireAt, want)
			}
		})
	}
}

// TestClientCredentials_TimeNowFallback proves instances built without
// the constructor still run on the real clock instead of panicking.
func TestClientCredentials_TimeNowFallback(t *testing.T) {
	var cc ClientCredentials

	before := time.Now()
	got := cc.timeNow()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("timeNow() = %v, want between %v and %v", got, before, after)
	}
}
