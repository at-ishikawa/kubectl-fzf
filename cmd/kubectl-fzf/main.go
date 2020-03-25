package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/at-ishikawa/kubectl-fzf/internal/command"
)

func main() {
	cli := cobra.Command{
		Use:   "kubectl-fzf [command]",
		Short: "kubectl commands with fzf",
	}
	commonFlags := cli.PersistentFlags()
	commonFlags.StringP("query", "q", "", "Start the fzf with this query")
	commonFlags.StringP("namespace", "n", "", "Kubernetes namespace")

	getCommand := cobra.Command{
		Use:   "get [resource]",
		Short: "kubectl get resources with fzf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[0]
			flags := cmd.Flags()
			namespace, err := flags.GetString("namespace")
			if err != nil {
				return err
			}
			previewFormat, err := flags.GetString("preview-format")
			if err != nil {
				return err
			}
			outputFormat, err := flags.GetString("output")
			if err != nil {
				return err
			}
			fzfQuery, err := flags.GetString("query")
			if err != nil {
				return err
			}

			cli, err := command.NewGetCli(resource, namespace, previewFormat, outputFormat, fzfQuery)
			if err != nil {
				return err
			}
			if err := cli.Run(context.Background(), os.Stdin, os.Stdout, os.Stderr); err != nil {
				return err
			}
			return nil
		},
	}
	getCommandFlags := getCommand.Flags()
	getCommandFlags.StringP("preview-format", "p", "describe", "The format of preview")
	getCommandFlags.StringP("output", "o", "name", "The output format")
	cli.AddCommand(&getCommand)

	describeCommand := cobra.Command{
		Use:   "describe [resource]",
		Short: "kubectl describe resources with fzf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[0]
			flags := cmd.Flags()
			namespace, err := flags.GetString("namespace")
			if err != nil {
				return err
			}
			fzfQuery, err := flags.GetString("query")
			if err != nil {
				return err
			}

			kubectl, err := command.NewKubectl(resource, namespace)
			if err != nil {
				return err
			}
			cli, err := command.NewDescribeCli(kubectl, fzfQuery)
			if err != nil {
				return err
			}
			if err := cli.Run(context.Background(), os.Stdin, os.Stdout, os.Stderr); err != nil {
				return err
			}
			return nil
		},
	}
	cli.AddCommand(&describeCommand)

	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
