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

type getCommand struct {
	resource       string
	previewCommand string
	outputFormat   string
}

const (
	kubectlOutputFormatDescribe = "describe"
	kubectlOutputFormatYaml     = "yaml"

	previewCommandDescribe = "kubectl describe {{ .resource }} {{ .name }}"
	previewCommandYaml     = "kubectl get {{ .resource }} {{ .name }} -o yaml"
)

var (
	errorInvalidArgumentResource       = errors.New("1st argument must be the kind of resources")
	errorInvalidArgumentPreviewCommand = errors.New("preview format must be one of [describe, yaml]")

	previewCommands = map[string]string{
		kubectlOutputFormatDescribe: previewCommandDescribe,
		kubectlOutputFormatYaml:     previewCommandYaml,
	}
)

func NewGetCommand(resource string, previewFormat string, outputFormat string) (*getCommand, error) {
	if resource == "" {
		return nil, errorInvalidArgumentResource
	}
	previewCommand, ok := previewCommands[previewFormat]
	if !ok {
		return nil, errorInvalidArgumentPreviewCommand
	}

	return &getCommand{
		resource:       resource,
		previewCommand: previewCommand,
		outputFormat:   outputFormat,
	}, nil
}

func (c getCommand) Run(ctx context.Context) (uint, error) {
	kubectlCommand := fmt.Sprintf("kubectl get %s", c.resource)
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
	fzfCommandLine := fmt.Sprintf("fzf --inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind ctrl-k:kill-line,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down", previewCommand)
	commandLine := fmt.Sprintf("%s | %s", kubectlCommand, fzfCommandLine)

	cmd := exec.CommandContext(ctx, "sh", "-c", commandLine)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 1, fmt.Errorf("failed to run a get: %w", err)
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
