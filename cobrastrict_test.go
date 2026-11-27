package cobrastrict

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newLockTree is a root with a lock group of two commands, and a replicate
// command that runs by itself and has a list subcommand.
func newLockTree() (root *cobra.Command, replicateRuns *int) {
	root = &cobra.Command{Use: "app", SilenceErrors: true, SilenceUsage: true}
	lock := &cobra.Command{Use: "lock", Short: "Show and break write locks"}
	for _, name := range []string{"show", "break"} {
		lock.AddCommand(&cobra.Command{
			Use:  name,
			RunE: func(command *cobra.Command, arguments []string) error { return nil },
		})
	}
	runs := 0
	replicate := &cobra.Command{
		Use:  "replicate <NAME>",
		Args: cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			runs++
			return nil
		},
	}
	replicate.AddCommand(&cobra.Command{
		Use:  "list",
		RunE: func(command *cobra.Command, arguments []string) error { return nil },
	})
	root.AddCommand(lock, replicate)
	RequireKnown(root)
	return root, &runs
}

// executeLockTree runs args against a fresh tree, returning its help output.
func executeLockTree(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root, _ := newLockTree()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	_, err := ExecuteContextC(t.Context(), root)
	return out.String(), err
}

// requireContains fails unless text contains every part.
func requireContains(t *testing.T, text string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(text, part) {
			t.Fatalf("%q does not contain %q", text, part)
		}
	}
}

// TestUnknownSubcommandFails checks that a group refuses a command it does
// not have, naming its commands and a near match.
func TestUnknownSubcommandFails(t *testing.T) {
	_, err := executeLockTree(t, "lock", "list")
	unknown, ok := errors.AsType[*UnknownCommandError](err)
	if !ok {
		t.Fatalf("lock list: %v", err)
	}
	if !errors.Is(err, ErrUnknownCommand) || unknown.Name != "list" || unknown.Command.Name() != "lock" {
		t.Fatalf("lock list: %+v", unknown)
	}
	if want := `unknown command "list" for "app lock"; its commands are break, show`; err.Error() != want {
		t.Fatalf("message %q, want %q", err, want)
	}

	_, err = executeLockTree(t, "lock", "shwo")
	requireContains(t, err.Error(), "did you mean show?")

	_, err = executeLockTree(t, "nosuch")
	requireContains(t, err.Error(), `unknown command "nosuch" for "app"`, "lock, replicate")
}

// TestGroupAloneShowsHelp checks that a group with no argument still prints
// its help and succeeds, without a usage line for flags it does not take.
func TestGroupAloneShowsHelp(t *testing.T) {
	help, err := executeLockTree(t, "lock")
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, help, "Show and break write locks", "app lock [command]")
	if strings.Contains(help, "app lock [flags]") {
		t.Fatalf("a group's help shows a [flags] usage line: %q", help)
	}
}

// TestRunnableCommandStillHasItsAction checks that a command with subcommands
// and an action of its own still runs it and shows its usage line.
func TestRunnableCommandStillHasItsAction(t *testing.T) {
	root, runs := newLockTree()
	root.SetArgs([]string{"replicate", "main.sqlite"})
	if err := root.Execute(); err != nil || *runs != 1 {
		t.Fatalf("replicate ran %d times: %v", *runs, err)
	}
	help, err := executeLockTree(t, "replicate", "--help")
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, help, "app replicate <NAME>")
}

// TestUnknownSubcommandHelpFails checks that --help after an unknown
// subcommand, and help with one, fail as the group does without them, while
// help on a known command prints it.
func TestUnknownSubcommandHelpFails(t *testing.T) {
	for _, arguments := range [][]string{
		{"nosuch", "--help"},
		{"--help", "nosuch"},
		{"lock", "list", "--help"},
		{"help", "nosuch"},
		{"help", "lock", "list"},
		{"help", "lock", "show", "extra"},
		{"help", "replicate", "nosuch"},
		{"help", "nosuch", "--help"},
		{"help", "lock", "list", "--help"},
	} {
		help, err := executeLockTree(t, arguments...)
		if !errors.Is(err, ErrUnknownCommand) {
			t.Fatalf("%v: %v", arguments, err)
		}
		requireContains(t, err.Error(), "unknown command")
		if help != "" {
			t.Fatalf("%v printed help %q", arguments, help)
		}
	}
	for _, arguments := range [][]string{
		{"help", "lock"},
		{"lock", "--help"},
		{"help"},
		{"help", "--help"},
		{"help", "lock", "--help"},
	} {
		help, err := executeLockTree(t, arguments...)
		if err != nil {
			t.Fatalf("%v: %v", arguments, err)
		}
		requireContains(t, help, "[command]")
	}
	help, err := executeLockTree(t, "help", "lock", "show")
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, help, "app lock show")
}

// TestHelpCompletesCommandPaths completes the commands of a group after
// help, as cobra's own help command does, and no file names.
func TestHelpCompletesCommandPaths(t *testing.T) {
	completions, err := executeLockTree(t, cobra.ShellCompRequestCmd, "help", "lock", "")
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, completions, "show", "break", ":4")

	completions, err = executeLockTree(t, cobra.ShellCompRequestCmd, "help", "h")
	if err != nil {
		t.Fatal(err)
	}
	requireContains(t, completions, "help\t", ":4")
}

// TestHelpDenialWithChildContext gives a group a context of its own, and
// runs one tree twice: --help on an unknown subcommand still fails each time.
func TestHelpDenialWithChildContext(t *testing.T) {
	root, _ := newLockTree()
	for _, child := range root.Commands() {
		child.SetContext(context.Background())
	}
	for run := range 2 {
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"lock", "bogus", "--help"})
		if _, err := ExecuteContextC(t.Context(), root); !errors.Is(err, ErrUnknownCommand) {
			t.Fatalf("run %d: lock bogus --help: %v", run, err)
		}
	}
}
