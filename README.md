# cobrastrict

[![Go Reference](https://pkg.go.dev/badge/github.com/jacob2161/cobrastrict.svg)](https://pkg.go.dev/github.com/jacob2161/cobrastrict)
[![CI](https://github.com/jacob2161/cobrastrict/actions/workflows/ci.yml/badge.svg)](https://github.com/jacob2161/cobrastrict/actions/workflows/ci.yml)

Strict command groups for cobra: a group fails on a subcommand it does not have (with a near-match
suggestion) instead of printing help and exiting 0, and help fails for unknown command paths.

```bash
go get github.com/jacob2161/cobrastrict
```

## Usage

```go
cobrastrict.RequireKnown(root) // once, after the tree is built
if _, err := cobrastrict.ExecuteContextC(ctx, root); err != nil {
	// app lock shwo: unknown command "shwo" for "app lock"; its commands are break, show; did you mean show?
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
```

- Cobra prints a group's help and exits 0 for `app lock list`, `app bogus --help`, and `app help
  bogus`, so a typo in a script passes unnoticed. After `RequireKnown`, all of them fail.
- Errors wrap `ErrUnknownCommand`; `*UnknownCommandError` carries the group, the argument, its
  available commands, and suggestions, so you can map it to your own exit code or error format.
- Shell completion after `help` still works.

Requires Go 1.27. Licensed under the MIT license.
