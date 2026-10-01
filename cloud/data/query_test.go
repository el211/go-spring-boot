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

package data

import "testing"

func TestParseMethod(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		subject Subject
		crit    []Criterion
		conns   []Connector
		orders  []Order
		limit   int
		bind    int
		wantErr bool
	}{
		{
			name:    "simple equality",
			method:  "FindByEmail",
			subject: SubjectFind,
			crit:    []Criterion{{Property: "email", Operator: OpEquals, Args: 1}},
			bind:    1,
		},
		{
			name:    "and of two",
			method:  "FindByEmailAndStatus",
			subject: SubjectFind,
			crit:    []Criterion{{Property: "email", Operator: OpEquals, Args: 1}, {Property: "status", Operator: OpEquals, Args: 1}},
			conns:   []Connector{ConnectorAnd},
			bind:    2,
		},
		{
			name:    "greater-than-equal wins over greater-than",
			method:  "FindByAgeGreaterThanEqual",
			subject: SubjectFind,
			crit:    []Criterion{{Property: "age", Operator: OpGreaterThanEqual, Args: 1}},
			bind:    1,
		},
		{
			name:    "between binds two",
			method:  "FindByCreatedAtBetween",
			subject: SubjectFind,
			crit:    []Criterion{{Property: "createdAt", Operator: OpBetween, Args: 2}},
			bind:    2,
		},
		{
			name:    "null binds zero",
			method:  "FindByDeletedAtIsNull",
			subject: SubjectFind,
			crit:    []Criterion{{Property: "deletedAt", Operator: OpIsNull, Args: 0}},
			bind:    0,
		},
		{
			name:    "order by with directions",
			method:  "FindByStatusOrderByCreatedAtDescName",
			subject: SubjectFind,
			crit:    []Criterion{{Property: "status", Operator: OpEquals, Args: 1}},
			orders:  []Order{{Property: "createdAt", Direction: Desc}, {Property: "name", Direction: Asc}},
			bind:    1,
		},
		{
			name:    "count subject",
			method:  "CountByStatus",
			subject: SubjectCount,
			crit:    []Criterion{{Property: "status", Operator: OpEquals, Args: 1}},
			bind:    1,
		},
		{
			name:    "exists subject",
			method:  "ExistsByEmail",
			subject: SubjectExists,
			crit:    []Criterion{{Property: "email", Operator: OpEquals, Args: 1}},
			bind:    1,
		},
		{
			name:    "delete subject",
			method:  "DeleteByStatus",
			subject: SubjectDelete,
			crit:    []Criterion{{Property: "status", Operator: OpEquals, Args: 1}},
			bind:    1,
		},
		{
			name:    "top-N limit",
			method:  "FindTop3ByStatusOrderByCreatedAtDesc",
			subject: SubjectFind,
			crit:    []Criterion{{Property: "status", Operator: OpEquals, Args: 1}},
			orders:  []Order{{Property: "createdAt", Direction: Desc}},
			limit:   3,
			bind:    1,
		},
		{
			name:    "or connector",
			method:  "FindByEmailOrPhone",
			subject: SubjectFind,
			crit:    []Criterion{{Property: "email", Operator: OpEquals, Args: 1}, {Property: "phone", Operator: OpEquals, Args: 1}},
			conns:   []Connector{ConnectorOr},
			bind:    2,
		},
		{
			name:    "find all no predicate",
			method:  "FindAll",
			subject: SubjectFind,
		},
		{
			name:    "no subject errors",
			method:  "LookupByEmail",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := ParseMethod(tt.method)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", q)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if q.Subject != tt.subject {
				t.Errorf("subject = %v, want %v", q.Subject, tt.subject)
			}
			if len(q.Criteria) != len(tt.crit) {
				t.Fatalf("criteria = %+v, want %+v", q.Criteria, tt.crit)
			}
			for i, c := range tt.crit {
				if q.Criteria[i] != c {
					t.Errorf("criterion[%d] = %+v, want %+v", i, q.Criteria[i], c)
				}
			}
			if len(q.Connectors) != len(tt.conns) {
				t.Errorf("connectors = %v, want %v", q.Connectors, tt.conns)
			} else {
				for i, c := range tt.conns {
					if q.Connectors[i] != c {
						t.Errorf("connector[%d] = %v, want %v", i, q.Connectors[i], c)
					}
				}
			}
			if len(q.Sort.Orders()) != len(tt.orders) {
				t.Fatalf("orders = %+v, want %+v", q.Sort.Orders(), tt.orders)
			}
			for i, o := range tt.orders {
				if q.Sort.Orders()[i] != o {
					t.Errorf("order[%d] = %+v, want %+v", i, q.Sort.Orders()[i], o)
				}
			}
			if q.Limit != tt.limit {
				t.Errorf("limit = %d, want %d", q.Limit, tt.limit)
			}
			if q.BindCount() != tt.bind {
				t.Errorf("bindCount = %d, want %d", q.BindCount(), tt.bind)
			}
		})
	}
}

func TestPageTotals(t *testing.T) {
	p := NewPage([]int{1, 2, 3}, PageRequest(0, 3, By("id")), 10)
	if got := p.TotalPages(); got != 4 {
		t.Errorf("TotalPages = %d, want 4", got)
	}
	if !p.HasNext() {
		t.Error("HasNext = false, want true")
	}
	last := NewPage([]int{10}, PageRequest(3, 3, By("id")), 10)
	if last.HasNext() {
		t.Error("last page HasNext = true, want false")
	}
}

func TestSortChaining(t *testing.T) {
	s := By("name").Descending().And(By("age"))
	orders := s.Orders()
	if len(orders) != 2 {
		t.Fatalf("orders = %+v", orders)
	}
	if orders[0] != (Order{Property: "name", Direction: Desc}) {
		t.Errorf("orders[0] = %+v", orders[0])
	}
	if orders[1] != (Order{Property: "age", Direction: Asc}) {
		t.Errorf("orders[1] = %+v", orders[1])
	}
}
