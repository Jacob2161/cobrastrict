// Package cobrastrict makes cobra command groups refuse subcommands they do
// not have.
//
// Cobra prints a group's help and exits 0 for "app lock list" when the
// command is "app lock show", for "app bogus --help", and for "app help
// bogus", so a typo in a script passes unnoticed. After RequireKnown, each of
// these fails with an *UnknownCommandError that names the group's commands and
// a near match, and ExecuteContextC returns it even when --help was given.
package cobrastrict

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ErrUnknownCommand is wrapped by every error for an argument that names no
// command, including *UnknownCommandError.
var ErrUnknownCommand = errors.New("unknown command")

// UnknownCommandError reports an argument to a group that names none of its
// subcommands.
type UnknownCommandError struct {
	// Command is the group the argument was given to.
	Command *cobra.Command
	// Name is the argument.
	Name string
	// Available names the group's available subcommands.
	Available []string
	// Suggestions are the available subcommands close to Name.
	Suggestions []string
}

func (e *UnknownCommandError) Error() string {
	if len(e.Available) == 0 {
		return fmt.Sprintf("unknown command %q for %q, which has no subcommands", e.Name, e.Command.CommandPath())
	}
	message := fmt.Sprintf(
		"unknown command %q for %q; its commands are %s",
		e.Name,
		e.Command.CommandPath(),
		strings.Join(e.Available, ", "),
	)
	if len(e.Suggestions) > 0 {
		message += fmt.Sprintf("; did you mean %s?", strings.Join(e.Suggestions, " or "))
	}
	return message
}

// Unwrap returns ErrUnknownCommand.
func (e *UnknownCommandError) Unwrap() error { return ErrUnknownCommand }

// groupAnnotation marks a group RequireKnown made runnable, so the usage
// template leaves out the "[flags]" line cobra shows for runnable commands.
const groupAnnotation = "cobrastrict.group"

// RequireKnown walks the command tree from root. A group, a command with
// subcommands and no action of its own, prints its help when run alone and
// fails with an *UnknownCommandError that names its commands when given
// any other argument. A command with an action of its own keeps it. Call it
// once, after the whole tree is built.
func RequireKnown(root *cobra.Command) {
	root.SetUsageTemplate(strings.Replace(
		root.UsageTemplate(),
		"Usage:{{if .Runnable}}",
		`Usage:{{if and .Runnable (not (index .Annotations "`+groupAnnotation+`"))}}`,
		1,
	))
	markGroups(root)
	printHelp := root.HelpFunc()
	root.SetHelpFunc(func(command *cobra.Command, arguments []string) {
		denyUnknownHelp(command, arguments, printHelp)
	})
	root.SetHelpCommand(newHelpCommand())
}

// helpDenialKey is the context key of the helpDenial ExecuteContextC
// passes down.
type helpDenialKey struct{}

// helpDenial is the error a --help on a group with an unknown subcommand
// fails with. Cobra answers --help through the help function, which returns
// no error, so the function saves it here and ExecuteContextC returns it.
type helpDenial struct {
	command *cobra.Command
	err     error
}

// ExecuteContextC runs root as cobra.Command.ExecuteContextC does, and also
// fails when --help was given to a group with a subcommand it does not have.
func ExecuteContextC(ctx context.Context, root *cobra.Command) (*cobra.Command, error) {
	denial := &helpDenial{}
	command, err := root.ExecuteContextC(context.WithValue(ctx, helpDenialKey{}, denial))
	if err == nil && denial.err != nil {
		return denial.command, denial.err
	}
	return command, err
}

// denyUnknownHelp prints a command's help, unless an argument names no
// command: an argument to a group that names none of its subcommands, or a
// path given to help that leads nowhere. Then --help fails as the command
// would without it.
func denyUnknownHelp(
	command *cobra.Command,
	arguments []string,
	printHelp func(*cobra.Command, []string),
) {
	var err error
	switch {
	case command.Annotations[helpCommandAnnotation] != "":
		_, err = helpTarget(command.Root(), command.Flags().Args())
	case IsGroup(command):
		err = denyUnknownCommand(command, command.Flags().Args())
	}
	if err == nil {
		printHelp(command, arguments)
		return
	}
	denial, saved := rootContextDenial(command)
	if !saved {
		command.PrintErrln(err.Error())
		return
	}
	denial.command, denial.err = command, err
}

// helpCommandAnnotation marks the help command newHelpCommand makes.
const helpCommandAnnotation = "cobrastrict.help-command"

// newHelpCommand is `help [command]`, which prints a command's help and,
// unlike cobra's, fails for a command the tree does not have.
func newHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:         "help [command]",
		Short:       "Help about any command",
		Long:        "Help prints the help of any command, as --help does.",
		Annotations: map[string]string{helpCommandAnnotation: "true"},
		// Completes the command path, as cobra's help command does.
		ValidArgsFunction: func(
			command *cobra.Command,
			arguments []string,
			toComplete string,
		) ([]cobra.Completion, cobra.ShellCompDirective) {
			target, _, err := command.Root().Find(arguments)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var completions []cobra.Completion
			for _, subcommand := range target.Commands() {
				// IsAvailableCommand leaves out the help command, which help takes too.
				listed := subcommand.IsAvailableCommand() ||
					subcommand.Annotations[helpCommandAnnotation] != ""
				if listed && strings.HasPrefix(subcommand.Name(), toComplete) {
					completions = append(
						completions,
						cobra.CompletionWithDesc(subcommand.Name(), subcommand.Short),
					)
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(command *cobra.Command, arguments []string) error {
			target, err := helpTarget(command.Root(), arguments)
			if err != nil {
				return err
			}
			target.InitDefaultHelpFlag()
			target.InitDefaultVersionFlag()
			return target.Help()
		},
	}
}

// helpTarget is the command a help path names, or an error wrapping
// ErrUnknownCommand when a word of the path names no command.
func helpTarget(root *cobra.Command, path []string) (*cobra.Command, error) {
	target, remaining, err := root.Find(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnknownCommand, err)
	}
	if len(remaining) > 0 {
		return nil, denyUnknownCommand(target, remaining)
	}
	return target, nil
}

// markGroups gives command and every group below it the group action.
func markGroups(command *cobra.Command) {
	for _, subcommand := range command.Commands() {
		markGroups(subcommand)
	}
	if !command.HasSubCommands() || command.Runnable() {
		return
	}
	if command.Annotations == nil {
		command.Annotations = map[string]string{}
	}
	command.Annotations[groupAnnotation] = "true"
	if command.SuggestionsMinimumDistance <= 0 {
		// Cobra's own default, which it applies only to the root command.
		command.SuggestionsMinimumDistance = 2
	}
	command.Args = denyUnknownCommand
	command.RunE = func(command *cobra.Command, arguments []string) error {
		return command.Help()
	}
}

// denyUnknownCommand fails for any argument to a group: an argument cobra
// did not match to a subcommand names none.
func denyUnknownCommand(command *cobra.Command, arguments []string) error {
	if len(arguments) == 0 {
		return nil
	}
	var names []string
	for _, subcommand := range command.Commands() {
		if subcommand.IsAvailableCommand() {
			names = append(names, subcommand.Name())
		}
	}
	unknown := &UnknownCommandError{Command: command, Name: arguments[0], Available: names}
	if len(names) > 0 {
		unknown.Suggestions = command.SuggestionsFor(arguments[0])
	}
	return unknown
}

// IsGroup reports whether RequireKnown gave command the group action.
func IsGroup(command *cobra.Command) bool {
	return command.Annotations[groupAnnotation] != ""
}

// rootContextDenial returns the helpDenial of the running ExecuteContextC.
// Cobra sets the root's context on every execution but a child's only when
// the child has none, so the root's is the current execution's.
func rootContextDenial(command *cobra.Command) (*helpDenial, bool) {
	ctx := command.Root().Context()
	if ctx == nil {
		return nil, false
	}
	denial, saved := ctx.Value(helpDenialKey{}).(*helpDenial)
	return denial, saved
}
