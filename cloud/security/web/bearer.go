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

// BearerAuthenticator builds an [Authenticator] that reads the Authorization:
// Bearer header and verifies it with v, the resource-server bridge to the
// existing [security.TokenValidator] seam (JWT, opaque-token introspection, …).
//
// A request with no bearer token is anonymous (nil, nil); a token v rejects is
// an invalid credential (nil, error) → 401. This is the OAuth2 resource-server
// filter in Spring terms, reusing the project's one validator abstraction.
func BearerAuthenticator(v security.TokenValidator) Authenticator {
	return func(r *http.Request) (*security.Authentication, error) {
		token := security.ParseBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			return nil, nil // anonymous
		}
		return v.Validate(r.Context(), token)
	}
}
