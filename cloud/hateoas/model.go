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

package hateoas

import (
	"encoding/json"
	"fmt"
)

// EntityModel wraps a single resource with links, the analog of
// org.springframework.hateoas.EntityModel<T>. In HAL the content's own fields
// are rendered at the top level, alongside the "_links" object.
type EntityModel[T any] struct {
	Content T
	RepresentationModel
}

// Entity wraps content and attaches the given links.
func Entity[T any](content T, links ...Link) *EntityModel[T] {
	e := &EntityModel[T]{Content: content}
	e.Add(links...)
	return e
}

// MarshalJSON renders HAL: the content's object fields are hoisted to the top
// level and "_links" is added. If the content does not marshal to a JSON
// object (e.g. a scalar or array), it is nested under a "content" key instead,
// since there is nothing to hoist.
func (e EntityModel[T]) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(e.Content)
	if err != nil {
		return nil, fmt.Errorf("hateoas: marshal content: %w", err)
	}
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		// Content is not a JSON object: nest it under "content".
		out := map[string]any{"content": e.Content}
		if links := e.linksHAL(); links != nil {
			out["_links"] = links
		}
		return json.Marshal(out)
	}
	merged := make(map[string]any, len(obj)+1)
	for k, v := range obj {
		merged[k] = v
	}
	if links := e.linksHAL(); links != nil {
		merged["_links"] = links
	}
	return json.Marshal(merged)
}

// CollectionModel wraps a list of resources with links, the analog of
// CollectionModel<T>. The items render under "content" and the collection's own
// links under "_links" (HAL's top-level collection shape).
type CollectionModel[T any] struct {
	Content []T
	RepresentationModel
}

// Collection wraps items and attaches the given links.
func Collection[T any](content []T, links ...Link) *CollectionModel[T] {
	c := &CollectionModel[T]{Content: content}
	c.Add(links...)
	return c
}

// MarshalJSON renders the collection as {"content":[...], "_links":{...}}.
func (c CollectionModel[T]) MarshalJSON() ([]byte, error) {
	out := map[string]any{"content": c.Content}
	if links := c.linksHAL(); links != nil {
		out["_links"] = links
	}
	return json.Marshal(out)
}
