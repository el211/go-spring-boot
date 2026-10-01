# cloud/shell — GoSpring Shell

The GoSpring port of **Spring Shell**: a registry of named commands driven by an
interactive read-eval-print loop. A [Command] is the analog of an
`@ShellMethod`; the loop reads from any `io.Reader` and writes to any
`io.Writer`, so it runs in a real terminal and is fully testable without one.

## Mapping to Spring Shell

| Spring Shell | `cloud/shell` |
|---|---|
| `@ShellComponent` / `@ShellMethod` | `Command{Name, Help, Run}` + `Register` |
| interactive shell | `Shell.Run(ctx, in, out)` |
| command parsing (quoted args) | quote-aware tokeniser |
| `help` built-in | built-in `help` command |
| `exit` / `quit` | built-in exit words |

## Example

```go
sh := shell.New("gs> ")
sh.Register(shell.Command{
    Name: "greet",
    Help: "Greet someone by name",
    Run: func(ctx context.Context, args []string, out io.Writer) error {
        fmt.Fprintf(out, "Hello, %s!\n", strings.Join(args, " "))
        return nil
    },
})
_ = sh.Run(ctx, os.Stdin, os.Stdout)
```

```
gs> greet "Ada Lovelace"
Hello, Ada Lovelace!
gs> help
Commands:
  greet   Greet someone by name
  help    List commands or show help for one
  exit    Leave the shell
gs> exit
```

## Semantics

- **Resilient loop.** A command error is printed and the loop continues; it ends
  only on `exit`/`quit` or EOF.
- **Quote-aware parsing.** `"hello world"` is a single argument.
- **Non-interactive use.** `Execute(ctx, line, out)` runs one line, for scripting
  or tests.

## Status

- [x] Command registry, quote-aware parsing, REPL, built-in help/exit
- [ ] Tab completion and command history
- [ ] Typed/validated argument binding (flags)
