package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{}

	return cmd
}

func Execute() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		w := cmd.ErrOrStderr()
		fmt.Fprintf(w, "Error: %+v\n", err)
		os.Exit(1)
	}
}
func init() {}
