# Changelog

This module follows semantic versioning. Before v1, a minor version may
change the API; each such change says what callers must do.

## Unreleased

- `UnknownCommandError.Error` has a doc comment. No API change.
- CI also checks formatting, `go mod tidy -diff`, and `govulncheck`.

## v0.1.0 (2026-11-27)

First release.

- `RequireKnown` makes every cobra command group refuse an argument that
  names none of its subcommands, with an `*UnknownCommandError` that lists
  the group's commands and a near match, instead of printing help and
  exiting 0.
- `help bogus` and `group bogus --help` fail the same way, and
  `ExecuteContextC` returns that failure even when `--help` was given.
