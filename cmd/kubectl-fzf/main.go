package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/at-ishikawa/kubectl-fzf/internal/command"
)

func main() {
	var exitCode int
	cli := cobra.Command{
		Use:   "kubectl-fzf [command]",
		Short: "kubectl commands with fzf",
	}
	getCommand := cobra.Command{
		Use:   "get [resource]",
		Short: "kubectl get resources with fzf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[0]
			previewFormat, err := cmd.Flags().GetString("preview-format")
			if err != nil {
				return err
			}
			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}

			co, err := command.NewGetCommand(resource, previewFormat, outputFormat)
			if err != nil {
				return err
			}

			exitCode, err = co.Run(context.Background(), os.Stdin, os.Stdout, os.Stderr)
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags := getCommand.Flags()
	flags.StringP("preview-format", "p", "describe", "The format of preview")
	flags.StringP("output", "o", "name", "The output format")

	cli.AddCommand(&getCommand)
	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}
