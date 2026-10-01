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

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// verbosity is the shared gs log level (see gs/CLAUDE.md): level 0 prints
// "[INFO] <label>" step lines, -v adds argv, -vv adds per-method detail.
var verbosity int

func splitVerbosity(args []string) (level int, rest []string) {
	for _, a := range args {
		switch {
		case a == "--verbose":
			level++
		case strings.HasPrefix(a, "--verbose="):
			if n, err := strconv.Atoi(a[len("--verbose="):]); err == nil {
				level += n
				continue
			}
			rest = append(rest, a)
		case len(a) >= 2 && a[0] == '-' && strings.Trim(a[1:], "v") == "":
			level += len(a) - 1
		default:
			rest = append(rest, a)
		}
	}
	return level, rest
}

func infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
}

func detailf(format string, args ...any) {
	if verbosity >= 2 {
		fmt.Fprintf(os.Stderr, "       "+format+"\n", args...)
	}
}
