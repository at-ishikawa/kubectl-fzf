package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/at-ishikawa/kubectl-fzf/internal/command"
)

func main() {
	cli := cobra.Command{
		Use:           "kubectl-fzf [resource]",
		Short:         "kubectl get [resource] command with fzf",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MinimumNArgs(1),
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
			fzfQuery, err := flags.GetString("query")
			if err != nil {
				return err
			}

			kubectl, err := command.NewKubectl(resource, namespace)
			if err != nil {
				return err
			}
			cli, err := command.NewGetCli(kubectl, previewFormat, fzfQuery)
			if err != nil {
				return err
			}
			if err := cli.Run(context.Background(), os.Stdin, os.Stdout, os.Stderr); err != nil {
				return err
			}
			return nil
		},
	}
	commonFlags := cli.Flags()
	commonFlags.StringP("query", "q", "", "Start the fzf with this query")
	commonFlags.StringP("namespace", "n", "", "Kubernetes namespace")
	commonFlags.StringP("preview-format", "p", "describe", "The format of preview")

	if err := cli.Execute(); err != nil {
		message := err.Error()
		if !strings.HasSuffix(message, "\n") {
			message = message + "\n"
		}
		_, werr := fmt.Fprint(os.Stderr, message)
		if werr != nil {
			fmt.Printf("failed to write the message %s on stderr", message)
		}
		os.Exit(1)
	}
	os.Exit(0)
}
