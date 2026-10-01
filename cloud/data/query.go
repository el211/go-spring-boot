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

import (
	"fmt"
	"strings"
)

// This file is the GoSpring port of Spring Data's query-derivation engine
// (org.springframework.data.repository.query.parser.PartTree). It turns a
// repository method name such as FindByEmailAndStatusOrderByCreatedAtDesc into
// a structured [Query]. Where Spring parses this reflectively at runtime and
// builds a proxy, the gospring-data generator calls [ParseMethod] at build
// time and emits a static method body — same name grammar, no reflection.

// Subject is the method's intent, parsed from its leading keyword, mirroring
// the "subject" of a Spring Data PartTree.
type Subject int

const (
	// SubjectFind is a read returning entities (Find.../Get.../Read.../Query...).
	SubjectFind Subject = iota
	// SubjectCount returns a row count (Count...).
	SubjectCount
	// SubjectExists returns a boolean (Exists...).
	SubjectExists
	// SubjectDelete removes rows (Delete.../Remove...).
	SubjectDelete
)

// Operator is a comparison applied to one property, mirroring the keyword set
// of PartTree.Part.Type.
type Operator int

const (
	OpEquals Operator = iota
	OpNotEquals
	OpLessThan
	OpLessThanEqual
	OpGreaterThan
	OpGreaterThanEqual
	OpLike
	OpNotLike
	OpContaining
	OpStartingWith
	OpEndingWith
	OpIn
	OpNotIn
	OpBetween
	OpIsNull
	OpIsNotNull
	OpTrue
	OpFalse
)

// bindCount is how many method arguments an operator consumes.
func (op Operator) bindCount() int {
	switch op {
	case OpIsNull, OpIsNotNull, OpTrue, OpFalse:
		return 0
	case OpBetween:
		return 2
	default:
		return 1
	}
}

// Criterion is one parsed predicate: a property, an operator and the number of
// method arguments it binds.
type Criterion struct {
	Property string
	Operator Operator
	Args     int
}

// Connector joins adjacent criteria, mirroring the And/Or keywords.
type Connector int

const (
	ConnectorAnd Connector = iota
	ConnectorOr
)

// Query is the structured form of a derived method name: the subject, the
// predicate (criteria joined by connectors) and any trailing OrderBy. Backends
// translate a Query to their native query language; the generator emits the
// code that calls a backend with this Query.
type Query struct {
	Subject    Subject
	Criteria   []Criterion
	Connectors []Connector // len == len(Criteria)-1; Connectors[i] joins Criteria[i] and [i+1]
	Sort       Sort
	Distinct   bool
	Limit      int // 0 means no limit; set by First/Top<N>
}

// BindCount is the total number of method arguments the predicate consumes —
// the arity the generator must supply beyond ctx.
func (q Query) BindCount() int {
	n := 0
	for _, c := range q.Criteria {
		n += c.Args
	}
	return n
}

// operatorKeywords are matched longest-first so GreaterThanEqual wins over
// GreaterThan. Order in this slice is the match precedence.
var operatorKeywords = []struct {
	suffix string
	op     Operator
}{
	{"IsNotNull", OpIsNotNull},
	{"NotNull", OpIsNotNull},
	{"IsNull", OpIsNull},
	{"Null", OpIsNull},
	{"GreaterThanEqual", OpGreaterThanEqual},
	{"GreaterThan", OpGreaterThan},
	{"LessThanEqual", OpLessThanEqual},
	{"LessThan", OpLessThan},
	{"NotLike", OpNotLike},
	{"Like", OpLike},
	{"StartingWith", OpStartingWith},
	{"EndingWith", OpEndingWith},
	{"Containing", OpContaining},
	{"NotIn", OpNotIn},
	{"In", OpIn},
	{"Between", OpBetween},
	{"IsTrue", OpTrue},
	{"True", OpTrue},
	{"IsFalse", OpFalse},
	{"False", OpFalse},
	{"Not", OpNotEquals},
	{"Is", OpEquals},
	{"Equals", OpEquals},
}

// ParseMethod parses a repository method name into a [Query], mirroring
// Spring's PartTree constructor. It is grammar-only: it does not check the
// properties against an entity (the generator does that against the struct).
func ParseMethod(name string) (Query, error) {
	var q Query

	rest, ok := cutSubject(name, &q)
	if !ok {
		return q, fmt.Errorf("data: method %q has no Find/Count/Exists/Delete subject", name)
	}

	// First<N>/Top<N> and Distinct limit keywords sit between subject and By.
	rest = parseLimitAndDistinct(rest, &q)

	// Split predicate from trailing OrderBy.
	predicate := rest
	if i := strings.Index(rest, "OrderBy"); i >= 0 {
		predicate = rest[:i]
		sort, err := parseOrderBy(rest[i+len("OrderBy"):])
		if err != nil {
			return q, err
		}
		q.Sort = sort
	}

	predicate = strings.TrimPrefix(predicate, "By")
	predicate = strings.TrimPrefix(predicate, "All") // FindAll with no predicate
	if predicate == "" {
		return q, nil
	}

	if err := parsePredicate(predicate, &q); err != nil {
		return q, err
	}
	return q, nil
}

func cutSubject(name string, q *Query) (string, bool) {
	// Longest/most-specific prefixes first.
	prefixes := []struct {
		p string
		s Subject
	}{
		{"Find", SubjectFind}, {"Get", SubjectFind}, {"Read", SubjectFind}, {"Query", SubjectFind},
		{"Count", SubjectCount},
		{"Exists", SubjectExists},
		{"Delete", SubjectDelete}, {"Remove", SubjectDelete},
	}
	for _, pr := range prefixes {
		if strings.HasPrefix(name, pr.p) {
			q.Subject = pr.s
			return name[len(pr.p):], true
		}
	}
	return name, false
}

func parseLimitAndDistinct(rest string, q *Query) string {
	if strings.HasPrefix(rest, "Distinct") {
		q.Distinct = true
		rest = rest[len("Distinct"):]
	}
	for _, kw := range []string{"First", "Top"} {
		if strings.HasPrefix(rest, kw) {
			after := rest[len(kw):]
			n, consumed := leadingInt(after)
			if consumed == 0 {
				q.Limit = 1 // bare "First"/"Top" means 1
			} else {
				q.Limit = n
				rest = after[consumed:]
				return rest
			}
			rest = after
		}
	}
	return rest
}

func leadingInt(s string) (val, consumed int) {
	for consumed < len(s) && s[consumed] >= '0' && s[consumed] <= '9' {
		val = val*10 + int(s[consumed]-'0')
		consumed++
	}
	return val, consumed
}

// parsePredicate splits on And/Or at property boundaries (a connector is only a
// connector when followed by an uppercase letter starting the next property).
func parsePredicate(predicate string, q *Query) error {
	segments, connectors := splitConnectors(predicate)
	q.Connectors = connectors
	for _, seg := range segments {
		c, err := parseCriterion(seg)
		if err != nil {
			return err
		}
		q.Criteria = append(q.Criteria, c)
	}
	return nil
}

func splitConnectors(s string) (segments []string, connectors []Connector) {
	start := 0
	i := 0
	for i < len(s) {
		if conn, kw, ok := connectorAt(s, i); ok {
			segments = append(segments, s[start:i])
			connectors = append(connectors, conn)
			i += len(kw)
			start = i
			continue
		}
		i++
	}
	segments = append(segments, s[start:])
	return segments, connectors
}

func connectorAt(s string, i int) (Connector, string, bool) {
	for _, kw := range []struct {
		word string
		conn Connector
	}{{"And", ConnectorAnd}, {"Or", ConnectorOr}} {
		if strings.HasPrefix(s[i:], kw.word) {
			next := i + len(kw.word)
			if next < len(s) && s[next] >= 'A' && s[next] <= 'Z' {
				return kw.conn, kw.word, true
			}
		}
	}
	return 0, "", false
}

func parseCriterion(seg string) (Criterion, error) {
	if seg == "" {
		return Criterion{}, fmt.Errorf("data: empty predicate segment")
	}
	for _, kw := range operatorKeywords {
		if strings.HasSuffix(seg, kw.suffix) && len(seg) > len(kw.suffix) {
			prop := seg[:len(seg)-len(kw.suffix)]
			return Criterion{Property: lowerFirst(prop), Operator: kw.op, Args: kw.op.bindCount()}, nil
		}
	}
	// No operator keyword: implicit equality on the whole property.
	return Criterion{Property: lowerFirst(seg), Operator: OpEquals, Args: 1}, nil
}

// parseOrderBy parses the trailing OrderBy clause, e.g. "CreatedAtDescNameAsc"
// into [createdAt DESC, name ASC]. A property with no explicit Asc/Desc suffix
// (only possible as the final property) defaults to ascending.
func parseOrderBy(s string) (Sort, error) {
	var sort Sort
	for s != "" {
		idxAsc := indexBoundary(s, "Asc")
		idxDesc := indexBoundary(s, "Desc")
		switch {
		case idxDesc >= 0 && (idxAsc < 0 || idxDesc < idxAsc):
			sort = sort.And(order(s[:idxDesc], Desc))
			s = s[idxDesc+len("Desc"):]
		case idxAsc >= 0:
			sort = sort.And(order(s[:idxAsc], Asc))
			s = s[idxAsc+len("Asc"):]
		default:
			// Trailing property with no explicit direction: ascending.
			sort = sort.And(order(s, Asc))
			s = ""
		}
	}
	if !sort.IsSorted() {
		return sort, fmt.Errorf("data: empty OrderBy clause")
	}
	return sort, nil
}

func order(prop string, dir Direction) Sort {
	return Sort{orders: []Order{{Property: lowerFirst(prop), Direction: dir}}}
}

// indexBoundary finds the first occurrence of kw that ends on a property
// boundary (end of string or an uppercase letter), so "Ascent" is not mistaken
// for the "Asc" keyword.
func indexBoundary(s, kw string) int {
	from := 0
	for {
		i := strings.Index(s[from:], kw)
		if i < 0 {
			return -1
		}
		abs := from + i
		if isBoundary(s, abs+len(kw)) {
			return abs
		}
		from = abs + len(kw)
	}
}

func isBoundary(s string, pos int) bool {
	return pos >= len(s) || (s[pos] >= 'A' && s[pos] <= 'Z')
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
