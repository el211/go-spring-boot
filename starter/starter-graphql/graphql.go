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

// Package gql is the GoSpring port of Spring GraphQL: a thin, Spring-flavoured
// layer over the graphql-go engine. You register query and mutation fields with
// resolver functions — the @QueryMapping / @MutationMapping analog — and get a
// built schema and an HTTP endpoint. The engine does parsing, validation and
// execution; this package provides the controller-style programming model and
// the net/http transport.
package gql

import (
	"context"

	"github.com/graphql-go/graphql"
)

// Resolver fetches the value for a field, the analog of a Spring @QueryMapping
// method / DataFetcher. args holds the field's GraphQL arguments; source is the
// parent object (nil for top-level query/mutation fields).
type Resolver func(ctx context.Context, args map[string]any, source any) (any, error)

// Field declares one query or mutation field and how to resolve it.
type Field struct {
	Name        string
	Type        graphql.Output
	Description string
	Args        graphql.FieldConfigArgument
	Resolve     Resolver
}

// Builder accumulates query and mutation fields, then produces a schema and an
// HTTP handler — the analog of assembling Spring GraphQL controllers into a
// GraphQlSource.
type Builder struct {
	queries   graphql.Fields
	mutations graphql.Fields
}

// New returns an empty [Builder].
func New() *Builder {
	return &Builder{queries: graphql.Fields{}, mutations: graphql.Fields{}}
}

// Query registers a top-level query field (@QueryMapping).
func (b *Builder) Query(f Field) *Builder {
	b.queries[f.Name] = b.toField(f)
	return b
}

// Mutation registers a top-level mutation field (@MutationMapping).
func (b *Builder) Mutation(f Field) *Builder {
	b.mutations[f.Name] = b.toField(f)
	return b
}

// toField adapts a [Field] to a graphql-go field definition, bridging the
// resolver signature to the engine's ResolveParams.
func (b *Builder) toField(f Field) *graphql.Field {
	gf := &graphql.Field{
		Name:        f.Name,
		Type:        f.Type,
		Description: f.Description,
		Args:        f.Args,
	}
	if f.Resolve != nil {
		gf.Resolve = func(p graphql.ResolveParams) (any, error) {
			return f.Resolve(p.Context, p.Args, p.Source)
		}
	}
	return gf
}

// Schema builds the executable schema. It errors if neither a query nor a
// mutation field was registered (GraphQL requires at least a query root).
func (b *Builder) Schema() (graphql.Schema, error) {
	cfg := graphql.SchemaConfig{}
	if len(b.queries) > 0 {
		cfg.Query = graphql.NewObject(graphql.ObjectConfig{Name: "Query", Fields: b.queries})
	}
	if len(b.mutations) > 0 {
		cfg.Mutation = graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: b.mutations})
	}
	return graphql.NewSchema(cfg)
}
