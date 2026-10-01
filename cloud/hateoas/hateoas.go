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

// Package hateoas is the GoSpring port of Spring HATEOAS: it adds hypermedia
// links to API resources and serialises them in HAL (the "_links" convention).
// [EntityModel] wraps a single resource, [CollectionModel] a list; both embed
// [RepresentationModel], which holds the links and renders the HAL "_links"
// object. The package is transport-neutral — it produces JSON, not routes.
package hateoas

// Standard IANA link relations, the common subset Spring exposes as constants.
const (
	RelSelf  = "self"
	RelNext  = "next"
	RelPrev  = "prev"
	RelFirst = "first"
	RelLast  = "last"
	RelItem  = "item"
)

// Link is a single hypermedia link, the analog of
// org.springframework.hateoas.Link. Rel is the relation name (the HAL key);
// the remaining fields render inside the link object.
type Link struct {
	Rel       string `json:"-"`
	Href      string `json:"href"`
	Templated bool   `json:"templated,omitempty"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
}

// NewLink builds a link for rel pointing at href.
func NewLink(rel, href string) Link { return Link{Rel: rel, Href: href} }

// Self is shorthand for the ubiquitous self link.
func Self(href string) Link { return Link{Rel: RelSelf, Href: href} }

// WithType sets the media type the link points to.
func (l Link) WithType(t string) Link { l.Type = t; return l }

// WithTitle sets a human-readable title.
func (l Link) WithTitle(t string) Link { l.Title = t; return l }

// Templated marks the href as a URI template (e.g. "/users/{id}").
func (l Link) AsTemplated() Link { l.Templated = true; return l }

// RepresentationModel is the link-carrying base embedded by the models, the
// analog of Spring's RepresentationModel. Embed it to make any type linkable.
type RepresentationModel struct {
	links []Link
}

// Add appends links to the model and returns the receiver for chaining.
func (r *RepresentationModel) Add(links ...Link) *RepresentationModel {
	r.links = append(r.links, links...)
	return r
}

// Links returns the model's links in insertion order.
func (r *RepresentationModel) Links() []Link { return r.links }

// GetLink returns the first link with the given relation; ok is false when
// none is present.
func (r *RepresentationModel) GetLink(rel string) (Link, bool) {
	for _, l := range r.links {
		if l.Rel == rel {
			return l, true
		}
	}
	return Link{}, false
}

// linksHAL renders the links as a HAL "_links" value: a map from relation to a
// single link object, or to an array when a relation appears more than once —
// exactly the shape Spring's HAL serialiser emits. It returns nil when there
// are no links, so "_links" is omitted entirely.
func (r *RepresentationModel) linksHAL() map[string]any {
	if len(r.links) == 0 {
		return nil
	}
	byRel := map[string][]Link{}
	order := []string{}
	for _, l := range r.links {
		if _, seen := byRel[l.Rel]; !seen {
			order = append(order, l.Rel)
		}
		byRel[l.Rel] = append(byRel[l.Rel], l)
	}
	out := make(map[string]any, len(order))
	for _, rel := range order {
		if ls := byRel[rel]; len(ls) == 1 {
			out[rel] = ls[0]
		} else {
			out[rel] = ls
		}
	}
	return out
}
