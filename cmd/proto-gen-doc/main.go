package main

import (
	"github.com/spf13/cobra"

	cmdbuild "github.com/smzgl/proto-gen-doc/internal/build"
)

// GetRootCommand is used to get root command
func GetRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "[proto-gen-doc]",
		Short: "proto-gen-doc is used to build markdown doc related proto files",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}

func main() {
	cmdRoot := GetRootCommand()
	cmdRoot.AddCommand(
		cmdbuild.CommandBuild(),
	)

	_ = cmdRoot.Execute()
}
