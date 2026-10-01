/*
 * Copyright 2025 The Go-Spring Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-spring.org/cloud/security"
)

// tokens maps a bearer token to the authentication it yields; unknown tokens
// are rejected.
type fakeValidator map[string]*security.Authentication

func (f fakeValidator) Validate(_ context.Context, token string) (*security.Authentication, error) {
	if a, ok := f[token]; ok {
		return a, nil
	}
	return nil, errors.New("invalid token")
}

func chain() http.Handler {
	v := fakeValidator{
		"admin": {Authenticated: true, Authorities: []string{"ROLE_ADMIN"}},
		"user":  {Authenticated: true, Authorities: []string{"ROLE_USER"}},
	}
	ok := func(w http.ResponseWriter, r *http.Request) {
		auth, _ := security.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		if auth != nil {
			_, _ = w.Write([]byte(auth.Principal.Subject))
		}
	}
	return NewFilterChain().
		Authentication(BearerAuthenticator(v)).
		Authorize(
			Matcher("/public/**").PermitAll(),
			Matcher("/admin/**").HasRole("ADMIN"),
			Matcher("/api/**").Authenticated(),
			AnyRequest().DenyAll(),
		).
		ThenFunc(ok)
}

func do(t *testing.T, h http.Handler, path, token string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestPermitAll(t *testing.T) {
	if code := do(t, chain(), "/public/info", ""); code != http.StatusOK {
		t.Fatalf("public anonymous = %d", code)
	}
}

func TestAuthenticatedRequiresToken(t *testing.T) {
	h := chain()
	if code := do(t, h, "/api/data", ""); code != http.StatusUnauthorized {
		t.Fatalf("anonymous /api = %d, want 401", code)
	}
	if code := do(t, h, "/api/data", "user"); code != http.StatusOK {
		t.Fatalf("user /api = %d, want 200", code)
	}
}

func TestRoleEnforcement(t *testing.T) {
	h := chain()
	if code := do(t, h, "/admin/panel", "user"); code != http.StatusForbidden {
		t.Fatalf("user /admin = %d, want 403", code)
	}
	if code := do(t, h, "/admin/panel", "admin"); code != http.StatusOK {
		t.Fatalf("admin /admin = %d, want 200", code)
	}
}

func TestInvalidTokenIs401(t *testing.T) {
	if code := do(t, chain(), "/api/data", "bogus"); code != http.StatusUnauthorized {
		t.Fatalf("bogus token = %d, want 401", code)
	}
}

func TestAnyRequestDenyAll(t *testing.T) {
	// /other matches only AnyRequest().DenyAll().
	if code := do(t, chain(), "/other", "admin"); code != http.StatusForbidden {
		t.Fatalf("denyAll authenticated = %d, want 403", code)
	}
	if code := do(t, chain(), "/other", ""); code != http.StatusUnauthorized {
		t.Fatalf("denyAll anonymous = %d, want 401", code)
	}
}

func TestNoMatchingRuleDenies(t *testing.T) {
	h := NewFilterChain().Authorize(Matcher("/x").PermitAll()).ThenFunc(
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	if code := do(t, h, "/unmatched", ""); code != http.StatusUnauthorized {
		t.Fatalf("unmatched path = %d, want deny", code)
	}
}
