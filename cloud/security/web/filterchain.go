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
	"net/http"

	"go-spring.org/cloud/security"
)

// Authenticator extracts and verifies the caller's identity from a request,
// the analog of an authentication filter. It returns (nil, nil) for an
// anonymous request (no credential), a non-nil [security.Authentication] for a
// verified one, and a non-nil error for a credential that was present but
// invalid — which the chain maps to 401.
type Authenticator func(r *http.Request) (*security.Authentication, error)

// FilterChain authenticates then authorises each request, the analog of a
// Spring SecurityFilterChain built with HttpSecurity. Build it fluently and
// wrap your handler with [FilterChain.Then].
type FilterChain struct {
	authenticator Authenticator
	rules         []AccessRule
	onDenied      func(w http.ResponseWriter, r *http.Request, authenticated bool)
}

// NewFilterChain returns an empty chain. With no rules every request is denied,
// so configure [FilterChain.Authorize] (ending in [AnyRequest]) explicitly.
func NewFilterChain() *FilterChain {
	return &FilterChain{onDenied: defaultDenied}
}

// Authentication sets the authenticator (e.g. [BearerAuthenticator]).
func (c *FilterChain) Authentication(a Authenticator) *FilterChain {
	c.authenticator = a
	return c
}

// Authorize sets the ordered access rules; the first whose pattern matches the
// request path decides, so put the most specific rules first and [AnyRequest]
// last.
func (c *FilterChain) Authorize(rules ...AccessRule) *FilterChain {
	c.rules = rules
	return c
}

// OnDenied overrides the response written when a request is rejected. The
// authenticated flag distinguishes a 403 (authenticated but unauthorised) from
// a 401 (anonymous).
func (c *FilterChain) OnDenied(fn func(w http.ResponseWriter, r *http.Request, authenticated bool)) *FilterChain {
	c.onDenied = fn
	return c
}

// Then wraps next with the chain: authenticate → publish the Authentication on
// the context → authorise → delegate.
func (c *FilterChain) Then(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var auth *security.Authentication
		if c.authenticator != nil {
			a, err := c.authenticator(r)
			if err != nil {
				c.onDenied(w, r, false) // credential present but invalid → 401
				return
			}
			auth = a
		}
		if auth != nil {
			r = r.WithContext(security.WithAuthentication(r.Context(), auth))
		}
		if !c.authorize(r.URL.Path, auth) {
			c.onDenied(w, r, auth.HasAnyAuthority())
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ThenFunc is [FilterChain.Then] for an http.HandlerFunc.
func (c *FilterChain) ThenFunc(next http.HandlerFunc) http.Handler {
	return c.Then(next)
}

// authorize finds the first rule matching path and applies it. No matching rule
// is a denial — secure by default, matching modern Spring Security.
func (c *FilterChain) authorize(path string, auth *security.Authentication) bool {
	for _, rule := range c.rules {
		if matchPath(rule.pattern, path) {
			return rule.allows(auth)
		}
	}
	return false
}

// defaultDenied writes 401 for an anonymous caller and 403 for an authenticated
// one lacking the required authority, mirroring Spring's entry-point / denied
// split.
func defaultDenied(w http.ResponseWriter, _ *http.Request, authenticated bool) {
	if authenticated {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}
