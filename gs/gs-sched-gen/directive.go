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

// Command gs-sched-gen is the GoSpring @Scheduled generator: it reads an
// interface whose methods carry //schedule: directives and emits a Register
// function that binds each method to a cloud/scheduling trigger and schedules
// it — the build-time replacement for Spring's @Scheduled post-processor.
package main

import (
	"fmt"
	"strings"
)

// schedule is one parsed //schedule: directive.
type schedule struct {
	kind         string // "fixedRate" | "fixedDelay" | "cron"
	rate         string // duration literal (fixedRate/fixedDelay) — a Go expr
	cron         string // raw cron expression (cron)
	initialDelay string // duration literal or ""
	jitter       string // duration literal or ""
	name         string // job name override or ""
	timeout      string // duration literal or ""
}

// parseSchedule reads the //schedule: directive from a method's doc lines.
func parseSchedule(lines []string) (schedule, bool, error) {
	for _, ln := range lines {
		ln = strings.TrimSpace(strings.TrimPrefix(ln, "//"))
		if !strings.HasPrefix(ln, "schedule:") {
			continue
		}
		return parseScheduleFields(strings.TrimPrefix(ln, "schedule:"))
	}
	return schedule{}, false, nil
}

func parseScheduleFields(s string) (schedule, bool, error) {
	toks := splitArgs(s)
	if len(toks) == 0 {
		return schedule{}, false, fmt.Errorf("empty //schedule: directive")
	}
	var sc schedule
	for _, tok := range toks {
		k, v, ok := strings.Cut(tok, "=")
		if !ok {
			return schedule{}, false, fmt.Errorf("malformed //schedule: option %q", tok)
		}
		switch k {
		case "fixedRate", "fixedDelay":
			sc.kind, sc.rate = k, durationLiteral(v)
		case "cron":
			sc.kind, sc.cron = "cron", v
		case "initialDelay":
			sc.initialDelay = durationLiteral(v)
		case "jitter":
			sc.jitter = durationLiteral(v)
		case "timeout":
			sc.timeout = durationLiteral(v)
		case "name":
			sc.name = v
		default:
			return schedule{}, false, fmt.Errorf("unknown //schedule: option %q", k)
		}
	}
	if sc.kind == "" {
		return schedule{}, false, fmt.Errorf("//schedule: needs one of fixedRate=, fixedDelay=, cron=")
	}
	return sc, true, nil
}

// splitArgs tokenises on whitespace but keeps a double-quoted value (a cron
// expression) as one token, with the surrounding quotes stripped.
func splitArgs(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"':
			inQuote = !inQuote
		case (c == ' ' || c == '\t') && !inQuote:
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}

// durationLiteral turns "5s"/"500ms"/"2h" into a Go expression so the emitted
// code needs no runtime parse. Unrecognised input falls back to ParseDuration.
func durationLiteral(s string) string {
	for _, u := range []struct {
		suffix string
		expr   string
	}{{"ms", "time.Millisecond"}, {"s", "time.Second"}, {"m", "time.Minute"}, {"h", "time.Hour"}} {
		if n, ok := strings.CutSuffix(s, u.suffix); ok && isDigits(n) {
			return n + " * " + u.expr
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
