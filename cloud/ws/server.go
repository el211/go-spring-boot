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

package ws

import (
	"io"
	"net/http"
)

// Server returns an http.Handler exposing the dispatcher as a SOAP endpoint,
// the analog of Spring WS's MessageDispatcherServlet. It accepts POSTed SOAP
// envelopes and replies with text/xml; a Fault is returned with HTTP 500 as the
// SOAP spec requires.
func (d *Dispatcher) Server() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "SOAP endpoint requires POST", http.StatusMethodNotAllowed)
			return
		}
		reqBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respBytes, handleErr := d.Handle(r.Context(), reqBytes)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		if handleErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, _ = w.Write(respBytes)
	})
}
