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

package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

func newEcho() *Shell {
	s := New("")
	s.Register(Command{Name: "echo", Help: "Print the arguments", Run: func(_ context.Context, args []string, out io.Writer) error {
		fmt.Fprintln(out, strings.Join(args, " "))
		return nil
	}})
	return s
}

func TestExecuteDispatch(t *testing.T) {
	s := newEcho()
	var buf bytes.Buffer
	if _, err := s.Execute(context.Background(), `echo "hello world" there`, &buf); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "hello world there\n" {
		t.Fatalf("echo output = %q", got)
	}
}

func TestUnknownCommandStaysAlive(t *testing.T) {
	s := newEcho()
	var buf bytes.Buffer
	exit, err := s.Execute(context.Background(), "nope", &buf)
	if exit || err != nil {
		t.Fatalf("exit=%v err=%v", exit, err)
	}
	if !strings.Contains(buf.String(), "unknown command: nope") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestExitCommands(t *testing.T) {
	s := newEcho()
	for _, word := range []string{"exit", "quit"} {
		exit, _ := s.Execute(context.Background(), word, io.Discard)
		if !exit {
			t.Fatalf("%q did not exit", word)
		}
	}
}

func TestRunLoop(t *testing.T) {
	s := newEcho()
	in := strings.NewReader("echo hi\nbogus\nexit\necho never\n")
	var out bytes.Buffer
	if err := s.Run(context.Background(), in, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "hi\n") {
		t.Fatalf("missing echo output: %q", got)
	}
	if !strings.Contains(got, "unknown command: bogus") {
		t.Fatalf("missing unknown-command notice: %q", got)
	}
	if strings.Contains(got, "never") {
		t.Fatalf("loop ran commands after exit: %q", got)
	}
}

func TestHelpLists(t *testing.T) {
	s := newEcho()
	var buf bytes.Buffer
	if _, err := s.Execute(context.Background(), "help", &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "echo") || !strings.Contains(out, "help") || !strings.Contains(out, "exit") {
		t.Fatalf("help listing incomplete: %q", out)
	}
}
