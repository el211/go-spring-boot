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

// Package restdocs is the GoSpring port of Spring REST Docs: it turns a real
// HTTP exchange captured in a test into documentation snippets (curl request,
// HTTP request/response, and field tables) written to disk, so the docs are
// generated from tests that must pass — they cannot drift from the API.
package restdocs

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Capture is one request/response exchange to be documented.
type Capture struct {
	Method          string
	URL             string
	RequestHeaders  http.Header
	RequestBody     []byte
	Status          int
	ResponseHeaders http.Header
	ResponseBody    []byte
}

// FromRecorder builds a [Capture] from a request (with its body passed
// explicitly, since a handler has usually already consumed req.Body) and the
// httptest recorder the handler wrote to.
func FromRecorder(req *http.Request, reqBody []byte, rec *httptest.ResponseRecorder) Capture {
	res := rec.Result()
	body := rec.Body.Bytes()
	return Capture{
		Method:          req.Method,
		URL:             req.URL.String(),
		RequestHeaders:  req.Header.Clone(),
		RequestBody:     reqBody,
		Status:          res.StatusCode,
		ResponseHeaders: res.Header.Clone(),
		ResponseBody:    body,
	}
}

// Documenter writes snippet files under a base directory, the analog of Spring
// REST Docs' generated-snippets output. Each documented exchange gets its own
// sub-directory named after the call.
type Documenter struct {
	dir string
}

// New returns a Documenter writing under dir (created on demand).
func New(dir string) *Documenter { return &Documenter{dir: dir} }

// Option customises what [Documenter.Document] emits.
type Option func(*docConfig)

type docConfig struct {
	reqFields  []FieldDescriptor
	respFields []FieldDescriptor
}

// Document writes the snippet set for one exchange into <dir>/<name>/. It always
// emits curl-request, http-request and http-response; request/response field
// tables are emitted (and validated) when the matching option is supplied. A
// field-validation failure returns an error and no field snippet is written,
// matching Spring's "undocumented/ missing field fails the test" contract.
func (d *Documenter) Document(name string, c Capture, opts ...Option) error {
	var cfg docConfig
	for _, o := range opts {
		o(&cfg)
	}
	out := filepath.Join(d.dir, name)
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	files := map[string]string{
		"curl-request.md":  curlSnippet(c),
		"http-request.md":  httpRequestSnippet(c),
		"http-response.md": httpResponseSnippet(c),
	}
	if cfg.reqFields != nil {
		table, err := fieldTable(c.RequestBody, cfg.reqFields)
		if err != nil {
			return fmt.Errorf("restdocs %q request fields: %w", name, err)
		}
		files["request-fields.md"] = table
	}
	if cfg.respFields != nil {
		table, err := fieldTable(c.ResponseBody, cfg.respFields)
		if err != nil {
			return fmt.Errorf("restdocs %q response fields: %w", name, err)
		}
		files["response-fields.md"] = table
	}
	for fname, content := range files {
		if err := os.WriteFile(filepath.Join(out, fname), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func curlSnippet(c Capture) string {
	var b strings.Builder
	fmt.Fprintf(&b, "```bash\n$ curl '%s' -i -X %s", c.URL, c.Method)
	for _, k := range sortedKeys(c.RequestHeaders) {
		fmt.Fprintf(&b, " \\\n    -H '%s: %s'", k, c.RequestHeaders.Get(k))
	}
	if len(c.RequestBody) > 0 {
		fmt.Fprintf(&b, " \\\n    -d '%s'", string(c.RequestBody))
	}
	b.WriteString("\n```\n")
	return b.String()
}

func httpRequestSnippet(c Capture) string {
	var b strings.Builder
	b.WriteString("```http\n")
	fmt.Fprintf(&b, "%s %s HTTP/1.1\n", c.Method, c.URL)
	writeHeaders(&b, c.RequestHeaders)
	if len(c.RequestBody) > 0 {
		fmt.Fprintf(&b, "\n%s\n", c.RequestBody)
	}
	b.WriteString("```\n")
	return b.String()
}

func httpResponseSnippet(c Capture) string {
	var b strings.Builder
	b.WriteString("```http\n")
	fmt.Fprintf(&b, "HTTP/1.1 %d %s\n", c.Status, http.StatusText(c.Status))
	writeHeaders(&b, c.ResponseHeaders)
	if len(c.ResponseBody) > 0 {
		fmt.Fprintf(&b, "\n%s\n", bytes.TrimRight(c.ResponseBody, "\n"))
	}
	b.WriteString("```\n")
	return b.String()
}

func writeHeaders(b *strings.Builder, h http.Header) {
	for _, k := range sortedKeys(h) {
		fmt.Fprintf(b, "%s: %s\n", k, h.Get(k))
	}
}

func sortedKeys(h http.Header) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
