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

// Package web is the GoSpring port of Spring Security's HttpSecurity /
// SecurityFilterChain: a per-request filter that authenticates the caller,
// establishes the [security.Authentication] on the context, and authorises the
// request against an ordered set of path rules. It builds on the primitives in
// cloud/security (Authentication, the SecurityContext accessors, TokenValidator)
// rather than restating them, so the one programming model spans RPC guards and
// the HTTP filter chain.
package web

import (
	"strings"

	"go-spring.org/cloud/security"
)

// rolePrefix is the Spring convention prepended by HasRole, kept separate from
// HasAuthority which uses the authority string verbatim.
const rolePrefix = "ROLE_"

type decision int

const (
	permitAll decision = iota
	denyAll
	authenticated
	hasAuthority
)

// AccessRule binds a request-path pattern to an authorization decision, the
// analog of one line in authorizeHttpRequests(...).
type AccessRule struct {
	pattern     string
	decision    decision
	authorities []string // for hasAuthority: caller needs any one of these
}

// Matcher starts an access rule for an ant-style path pattern ("/admin/**").
// Terminate it with one of the decision methods.
func Matcher(pattern string) Matcher2 { return Matcher2{pattern: pattern} }

// AnyRequest starts a rule matching every path, the analog of anyRequest().
// Place it last — rules are evaluated in order and the first match wins.
func AnyRequest() Matcher2 { return Matcher2{pattern: "/**"} }

// Matcher2 is the half-built rule returned by [Matcher] / [AnyRequest].
type Matcher2 struct{ pattern string }

// PermitAll allows the request unconditionally.
func (m Matcher2) PermitAll() AccessRule {
	return AccessRule{pattern: m.pattern, decision: permitAll}
}

// DenyAll rejects the request unconditionally.
func (m Matcher2) DenyAll() AccessRule {
	return AccessRule{pattern: m.pattern, decision: denyAll}
}

// Authenticated requires a verified identity, with no particular authority.
func (m Matcher2) Authenticated() AccessRule {
	return AccessRule{pattern: m.pattern, decision: authenticated}
}

// HasAuthority requires the caller to carry at least one of the authorities.
func (m Matcher2) HasAuthority(authorities ...string) AccessRule {
	return AccessRule{pattern: m.pattern, decision: hasAuthority, authorities: authorities}
}

// HasRole is HasAuthority with the Spring "ROLE_" prefix applied to each role,
// so HasRole("ADMIN") matches the authority "ROLE_ADMIN".
func (m Matcher2) HasRole(roles ...string) AccessRule {
	auths := make([]string, len(roles))
	for i, r := range roles {
		auths[i] = rolePrefix + r
	}
	return AccessRule{pattern: m.pattern, decision: hasAuthority, authorities: auths}
}

// allows reports whether auth satisfies the rule.
func (r AccessRule) allows(auth *security.Authentication) bool {
	switch r.decision {
	case permitAll:
		return true
	case denyAll:
		return false
	case authenticated:
		return auth.HasAnyAuthority()
	default: // hasAuthority
		return auth.HasAnyAuthority(r.authorities...)
	}
}

// matchPath reports whether an ant-style pattern matches path. "*" matches one
// segment, "**" matches any number of trailing segments.
func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}
	p := strings.Split(strings.Trim(pattern, "/"), "/")
	d := strings.Split(strings.Trim(path, "/"), "/")
	for i := 0; i < len(p); i++ {
		if p[i] == "**" {
			return true
		}
		if i >= len(d) {
			return false
		}
		if p[i] != "*" && p[i] != d[i] {
			return false
		}
	}
	return len(p) == len(d)
}
