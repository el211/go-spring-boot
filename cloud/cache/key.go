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

package cache

import (
	"fmt"
	"strings"
)

// BuildKey composes a cache key from a cache name and the key parts, the
// GoSpring analog of Spring's default KeyGenerator. The name namespaces the
// entry so two caches never collide; the parts (typically a method's arguments)
// identify the entry within that cache. A method with no key parts caches a
// single entry under the bare name, matching Spring's SimpleKey.EMPTY.
//
//	BuildKey("books", isbn)        // "books::978-0135"
//	BuildKey("books", author, year) // "books::Tolkien::1954"
//	BuildKey("catalog")             // "catalog"
func BuildKey(name string, parts ...any) string {
	if len(parts) == 0 {
		return name
	}
	var b strings.Builder
	b.WriteString(name)
	for _, p := range parts {
		b.WriteString("::")
		b.WriteString(fmt.Sprint(p))
	}
	return b.String()
}
