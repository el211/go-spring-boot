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

// Package shell is the GoSpring port of Spring Shell: a registry of named
// commands driven by an interactive read-eval-print loop. A command is the
// analog of an @ShellMethod; registering one makes it callable by name with
// quote-aware argument parsing. The loop reads from any io.Reader and writes to
// any io.Writer, so it is fully testable without a terminal.
package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Command is one shell command, the analog of a method annotated @ShellMethod.
type Command struct {
	// Name is the word that invokes the command.
	Name string
	// Help is the one-line description shown by the built-in help command.
	Help string
	// Run executes the command with the parsed arguments (the command word
	// excluded) and writes any output to out.
	Run func(ctx context.Context, args []string, out io.Writer) error
}

// Shell holds the command registry and the interactive loop.
type Shell struct {
	cmds   map[string]Command
	order  []string
	prompt string
}

// New returns a Shell with the built-in help, exit and quit commands already
// registered. prompt is printed before each line ("" for no prompt).
func New(prompt string) *Shell {
	s := &Shell{cmds: map[string]Command{}, prompt: prompt}
	s.Register(Command{Name: "help", Help: "List commands or show help for one", Run: s.help})
	return s
}

// Register adds or replaces a command. The built-in names help/exit/quit may be
// overridden by registering the same name.
func (s *Shell) Register(c Command) {
	if _, exists := s.cmds[c.Name]; !exists {
		s.order = append(s.order, c.Name)
	}
	s.cmds[c.Name] = c
}

// Run drives the read-eval-print loop until an exit/quit command or EOF on in.
// A command error is reported to out and the loop continues, so one bad command
// does not end the session.
func (s *Shell) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	sc := bufio.NewScanner(in)
	for {
		if s.prompt != "" {
			fmt.Fprint(out, s.prompt)
		}
		if !sc.Scan() {
			return sc.Err()
		}
		exit, err := s.Execute(ctx, sc.Text(), out)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
		}
		if exit {
			return nil
		}
	}
}

// Execute runs a single input line. It returns exit=true for the built-in
// exit/quit commands. A blank line is a no-op; an unknown command writes a
// message to out and returns a nil error so the loop stays alive.
func (s *Shell) Execute(ctx context.Context, line string, out io.Writer) (exit bool, err error) {
	args := splitArgs(line)
	if len(args) == 0 {
		return false, nil
	}
	name, rest := args[0], args[1:]
	if name == "exit" || name == "quit" {
		return true, nil
	}
	c, ok := s.cmds[name]
	if !ok {
		fmt.Fprintf(out, "unknown command: %s (try \"help\")\n", name)
		return false, nil
	}
	return false, c.Run(ctx, rest, out)
}

// help is the built-in help command: with no argument it lists every command;
// with a command name it prints that command's help line.
func (s *Shell) help(_ context.Context, args []string, out io.Writer) error {
	if len(args) == 1 {
		if c, ok := s.cmds[args[0]]; ok {
			fmt.Fprintf(out, "%s\t%s\n", c.Name, c.Help)
			return nil
		}
		return fmt.Errorf("no such command: %s", args[0])
	}
	names := append([]string{}, s.order...)
	sort.Strings(names)
	fmt.Fprintln(out, "Commands:")
	for _, n := range names {
		fmt.Fprintf(out, "  %s\t%s\n", n, s.cmds[n].Help)
	}
	fmt.Fprintln(out, "  exit\tLeave the shell")
	return nil
}

// splitArgs tokenises a line on whitespace, honouring double-quoted spans so an
// argument may contain spaces ("hello world" is one token).
func splitArgs(line string) []string {
	var out []string
	var cur strings.Builder
	inQuote, has := false, false
	flush := func() {
		if has {
			out = append(out, cur.String())
			cur.Reset()
			has = false
		}
	}
	for i := 0; i < len(line); i++ {
		switch c := line[i]; {
		case c == '"':
			inQuote = !inQuote
			has = true
		case (c == ' ' || c == '\t') && !inQuote:
			flush()
		default:
			cur.WriteByte(c)
			has = true
		}
	}
	flush()
	return out
}
