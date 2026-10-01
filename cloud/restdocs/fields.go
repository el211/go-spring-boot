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

package restdocs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FieldDescriptor documents one JSON field, the analog of Spring's
// PayloadDocumentation.fieldWithPath. Optional fields are exempt from the
// "every field must be present" check.
type FieldDescriptor struct {
	Path        string
	Type        string
	Description string
	Optional    bool
}

// Field builds a required descriptor for a top-level JSON field.
func Field(path, description string) FieldDescriptor {
	return FieldDescriptor{Path: path, Description: description}
}

// WithType sets the documented type (e.g. "String", "Number").
func (f FieldDescriptor) WithType(t string) FieldDescriptor { f.Type = t; return f }

// AsOptional marks the field optional.
func (f FieldDescriptor) AsOptional() FieldDescriptor { f.Optional = true; return f }

// WithRequestFields documents (and validates) the request body's fields.
func WithRequestFields(fs ...FieldDescriptor) Option {
	return func(c *docConfig) { c.reqFields = fs }
}

// WithResponseFields documents (and validates) the response body's fields.
func WithResponseFields(fs ...FieldDescriptor) Option {
	return func(c *docConfig) { c.respFields = fs }
}

// fieldTable validates descriptors against the JSON body's top-level keys and
// renders a Markdown table. Validation is bidirectional, as in Spring REST
// Docs: every non-optional documented field must be present, and every actual
// field must be documented — either gap fails the test.
func fieldTable(body []byte, fields []FieldDescriptor) (string, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return "", fmt.Errorf("body is not a JSON object: %w", err)
	}
	documented := map[string]bool{}
	for _, f := range fields {
		documented[f.Path] = true
		if _, present := obj[f.Path]; !present && !f.Optional {
			return "", fmt.Errorf("documented field %q is not present in the payload", f.Path)
		}
	}
	var undocumented []string
	for k := range obj {
		if !documented[k] {
			undocumented = append(undocumented, k)
		}
	}
	if len(undocumented) > 0 {
		sort.Strings(undocumented)
		return "", fmt.Errorf("undocumented fields in payload: %s", strings.Join(undocumented, ", "))
	}
	return renderTable(fields), nil
}

func renderTable(fields []FieldDescriptor) string {
	var b strings.Builder
	b.WriteString("| Path | Type | Description |\n|---|---|---|\n")
	for _, f := range fields {
		typ := f.Type
		if f.Optional {
			typ = strings.TrimSpace(typ + " (optional)")
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", f.Path, typ, f.Description)
	}
	return b.String()
}
