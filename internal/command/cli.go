package command

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
)

type cli struct {
	resource       string
	previewCommand string
	outputFormat   string
}

func NewCli(args []string, previewFormat string, outputFormat string) (*cli, error) {
	resource := args[0]
	var previewCommand string
	switch previewFormat {
	case "describe":
		previewCommand = previewCommandDescribe
	case "yaml":
		previewCommand = previewCommandYaml
	default:
		return nil, errors.New("preview format must be one of [describe, yaml]")
	}

	return &cli{
		resource:       resource,
		previewCommand: previewCommand,
		outputFormat:   outputFormat,
	}, nil
}

const (
	previewCommandDescribe = "kubectl describe {{ .resource }} {{ .name }}"
	previewCommandYaml     = "kubectl get {{ .resource }} {{ .name }} -o yaml"
)

func (c cli) Run(ctx context.Context) (int, error) {
	kubectlCommand := fmt.Sprintf("kubectl get %s --no-headers", c.resource)
	tmpl, err := template.New("preview").Parse(c.previewCommand)
	if err != nil {
		return 1, fmt.Errorf("failed to parse preview command: %w", err)
	}
	builder := strings.Builder{}
	if err = tmpl.Execute(&builder, map[string]interface{}{
		"resource": c.resource,
		"name":     "{1}",
	}); err != nil {
		return 1, fmt.Errorf("failed to parse preview command: %w", err)
	}

	previewCommand := builder.String()
	fzfCommandLine := fmt.Sprintf("fzf --layout reverse --preview '%s' --preview-window down:80%% --bind ctrl-k:kill-line,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down", previewCommand)
	commandLine := fmt.Sprintf("%s | %s", kubectlCommand, fzfCommandLine)

	cmd := exec.CommandContext(ctx, "sh", "-c", commandLine)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 1, fmt.Errorf("failed to run a cli: %w", err)
	}

	line := strings.TrimSpace(string(out))
	columns := strings.Fields(line)
	name := strings.TrimSpace(columns[0])

	switch c.outputFormat {
	case "name":
		fmt.Println(name)
	case "yaml":
		args := []string{
			"get",
			c.resource,
			"-o",
			"yaml",
			name,
		}
		out, err := exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
		if err != nil {
			return 2, fmt.Errorf("failed to output: %w. Output command result: %s", err, string(out))
		}
		fmt.Print(string(out))
	case "json":
		args := []string{
			"get",
			c.resource,
			"-o",
			"json",
			name,
		}
		out, err := exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
		if err != nil {
			return 2, fmt.Errorf("failed to output: %w. Output command result: %s", err, string(out))
		}
		fmt.Print(string(out))
	case "describe":
		args := []string{
			"describe",
			c.resource,
			name,
		}
		out, err := exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
		if err != nil {
			return 2, fmt.Errorf("failed to output: %w. Output command result: %s", err, string(out))
		}
		fmt.Print(string(out))
	}
	return 0, nil
}
