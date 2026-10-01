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

package gql

import (
	"encoding/json"
	"net/http"

	"github.com/graphql-go/graphql"
)

// request is the standard GraphQL-over-HTTP POST body.
type request struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
}

// Handler builds the schema and returns an http.Handler serving GraphQL over
// HTTP (POST application/json), the analog of Spring GraphQL's HTTP endpoint.
// Execution errors are reported in the GraphQL "errors" array with HTTP 200, as
// the GraphQL-over-HTTP convention requires; only a malformed request is a 400.
func (b *Builder) Handler() (http.Handler, error) {
	schema, err := b.Schema()
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "GraphQL endpoint requires POST", http.StatusMethodNotAllowed)
			return
		}
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid GraphQL request body", http.StatusBadRequest)
			return
		}
		result := graphql.Do(graphql.Params{
			Schema:         schema,
			RequestString:  req.Query,
			OperationName:  req.OperationName,
			VariableValues: req.Variables,
			Context:        r.Context(),
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}), nil
}
