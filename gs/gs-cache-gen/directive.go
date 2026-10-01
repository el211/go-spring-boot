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

// Command gs-cache-gen is the GoSpring caching generator: the port of Spring's
// caching abstraction (@Cacheable / @CachePut / @CacheEvict). It reads an
// interface whose methods carry //cache: directives and emits a decorator that
// wraps an implementation with cache-aside logic over cloud/cache — a
// build-time replacement for Spring's AOP cache interceptor.
package main

import (
	"fmt"
	"strings"
)

// kind is the caching behaviour requested by a //cache: directive.
type kind int

const (
	cacheable kind = iota // read-through: serve from cache, else call and store
	put                   // write-through: always call, then store the result
	evict                 // invalidate the entry after the call
)

// directive is one parsed //cache: line.
type directive struct {
	kind kind
	name string   // cache name (namespace)
	key  []string // parameter names forming the key; empty means all non-ctx params
	ttl  string   // Go duration literal, e.g. "5 * time.Minute"; "0" means no expiry
	all  bool     // evict: clear the bare-name entry
}

// parseDirective reads the //cache: directive out of a method's doc comment
// lines, returning ok=false when the method carries none (it is then left
// as a plain pass-through).
func parseDirective(lines []string) (directive, bool, error) {
	for _, ln := range lines {
		ln = strings.TrimSpace(strings.TrimPrefix(ln, "//"))
		if !strings.HasPrefix(ln, "cache:") {
			continue
		}
		return parseFields(strings.TrimPrefix(ln, "cache:"))
	}
	return directive{}, false, nil
}

func parseFields(s string) (directive, bool, error) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return directive{}, false, fmt.Errorf("empty //cache: directive")
	}
	var d directive
	switch fields[0] {
	case "cacheable":
		d.kind = cacheable
	case "put":
		d.kind = put
	case "evict":
		d.kind = evict
	default:
		return directive{}, false, fmt.Errorf("unknown cache verb %q", fields[0])
	}
	d.ttl = "0"
	for _, f := range fields[1:] {
		switch {
		case f == "all":
			d.all = true
		case strings.HasPrefix(f, "name="):
			d.name = strings.TrimPrefix(f, "name=")
		case strings.HasPrefix(f, "key="):
			d.key = strings.Split(strings.TrimPrefix(f, "key="), ",")
		case strings.HasPrefix(f, "ttl="):
			d.ttl = durationLiteral(strings.TrimPrefix(f, "ttl="))
		default:
			return directive{}, false, fmt.Errorf("unknown cache option %q", f)
		}
	}
	if d.name == "" {
		return directive{}, false, fmt.Errorf("//cache: directive missing name=")
	}
	return d, true, nil
}

// durationLiteral turns a human TTL like "5m" into a Go expression the emitted
// code can use without a parse at runtime. Unrecognised input falls back to a
// ParseDuration call so nothing is silently dropped.
func durationLiteral(s string) string {
	units := map[byte]string{'s': "time.Second", 'm': "time.Minute", 'h': "time.Hour"}
	if len(s) >= 2 {
		if u, ok := units[s[len(s)-1]]; ok {
			if n := s[:len(s)-1]; isDigits(n) {
				return n + " * " + u
			}
		}
	}
	return fmt.Sprintf("func() time.Duration { d, _ := time.ParseDuration(%q); return d }()", s)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
