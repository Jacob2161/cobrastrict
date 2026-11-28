# Changelog

This module follows semantic versioning. Before v1, a minor version may
change the API; each such change says what callers must do.

## v0.2.1 (2026-11-28)

- CI runs every step in bash, so the Windows runners read
  `-modfile=tools/go.mod` whole. No API change.
- The README says the module requires Go 1.27.

## v0.2.0 (2026-11-28)

- Requires Go 1.27, the newest Go release; Go 1.26 is no longer
  supported.
- CI runs staticcheck and govulncheck at the versions pinned in
  `tools/go.mod`, a module of its own that importers never see.

## v0.1.1 (2026-11-27)

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
