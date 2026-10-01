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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"go-spring.org/cloud/modulith"
)

// config is the declarative module model the tool reads (modulith.json), the
// file-based analog of Spring's @ApplicationModule annotations.
type config struct {
	Modules []modulith.Module `json:"modules"`
}

// loadConfig reads and decodes the module declaration file.
func loadConfig(path string) (*modulith.Modules, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gs-modulith: read config: %w", err)
	}
	var c config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("gs-modulith: parse %s: %w", path, err)
	}
	if len(c.Modules) == 0 {
		return nil, fmt.Errorf("gs-modulith: %s declares no modules", path)
	}
	return modulith.New(c.Modules...), nil
}

// goListPackage is the slice of `go list -json` output this tool consumes.
type goListPackage struct {
	ImportPath string
	Imports    []string
}

// loadGraph runs `go list -json ./...` in dir and returns the import graph as
// [modulith.Package] values. Using the go toolchain keeps resolution exactly
// consistent with how the code actually builds, with no extra dependency.
func loadGraph(dir string) ([]modulith.Package, error) {
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gs-modulith: go list failed: %w\n%s", err, stderr.String())
	}

	var out []modulith.Package
	dec := json.NewDecoder(&stdout)
	for {
		var p goListPackage
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("gs-modulith: decode go list output: %w", err)
		}
		out = append(out, modulith.Package{ImportPath: p.ImportPath, Imports: p.Imports})
	}
	return out, nil
}
