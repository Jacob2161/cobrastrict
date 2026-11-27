package cobrastrict_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jacob2161/cobrastrict"
)

func ExampleRequireKnown() {
	root := &cobra.Command{Use: "app", SilenceErrors: true, SilenceUsage: true}
	lock := &cobra.Command{Use: "lock", Short: "Show and break write locks"}
	lock.AddCommand(
		&cobra.Command{Use: "show", RunE: func(*cobra.Command, []string) error { return nil }},
	)
	root.AddCommand(lock)
	cobrastrict.RequireKnown(root)

	root.SetArgs([]string{"lock", "shwo"})
	_, err := cobrastrict.ExecuteContextC(context.Background(), root)
	fmt.Println(errors.Is(err, cobrastrict.ErrUnknownCommand))
	fmt.Println(err)
	// Output:
	// true
	// unknown command "shwo" for "app lock"; its commands are show; did you mean show?
}
