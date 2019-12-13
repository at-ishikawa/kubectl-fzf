package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os/exec"
	"strings"
)

type getCommand struct {
	resource       string
	previewCommand string
	outputFormat   string
}

const (
	kubectlOutputFormatName     = "name"
	kubectlOutputFormatDescribe = "describe"
	kubectlOutputFormatYaml     = "yaml"
	kubectlOutputFormatJSON     = "json"

	previewCommandDescribe = "kubectl describe {{ .resource }} {{ .name }}"
	previewCommandYaml     = "kubectl get {{ .resource }} {{ .name }} -o yaml"
)

var (
	errorInvalidArgumentResource       = errors.New("1st argument must be the kind of resources")
	errorInvalidArgumentPreviewCommand = errors.New("preview format must be one of [describe, yaml]")
	errorInvalidArgumentOutputFormat   = errors.New("output format must be one of [name, describe, yaml, json]")

	runCommandWithFzf = func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "sh", "-c", commandLine)
		cmd.Stderr = ioErr
		cmd.Stdin = ioIn
		return cmd.Output()
	}
	runKubectl = func(ctx context.Context, args []string) ([]byte, error) {
		return exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
	}
	previewCommands = map[string]string{
		kubectlOutputFormatDescribe: previewCommandDescribe,
		kubectlOutputFormatYaml:     previewCommandYaml,
	}
	outputFormats = map[string]struct{}{
		kubectlOutputFormatName:     {},
		kubectlOutputFormatDescribe: {},
		kubectlOutputFormatYaml:     {},
		kubectlOutputFormatJSON:     {},
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
	if _, ok := outputFormats[outputFormat]; !ok {
		return nil, errorInvalidArgumentOutputFormat
	}

	return &getCommand{
		resource:       resource,
		previewCommand: previewCommand,
		outputFormat:   outputFormat,
	}, nil
}

func (c getCommand) Run(ctx context.Context, ioIn io.Reader, ioOut io.Writer, ioErr io.Writer) (int, error) {
	kubectlCommand := fmt.Sprintf("kubectl get %s", c.resource)
	previewCommand, err := buildCommand("preview", c.previewCommand, map[string]interface{}{
		"resource": c.resource,
		"name":     "{1}",
	})
	if err != nil {
		return 1, fmt.Errorf("invalid preview command: %w", err)
	}
	fzfCommandLine := fmt.Sprintf("fzf --inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind ctrl-k:kill-line,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down", previewCommand)
	commandLine := fmt.Sprintf("%s | %s", kubectlCommand, fzfCommandLine)

	out, err := runCommandWithFzf(ctx, commandLine, ioIn, ioErr)
	if err != nil {
		return 1, fmt.Errorf("failed to run a get: %w", err)
	}

	line := strings.TrimSpace(string(out))
	columns := strings.Fields(line)
	name := strings.TrimSpace(columns[0])

	if c.outputFormat == kubectlOutputFormatName {
		out = bytes.NewBufferString(name).Bytes()
	} else {
		var args []string
		if c.outputFormat == kubectlOutputFormatDescribe {
			args = []string{
				"describe",
				c.resource,
				name,
			}
		} else if c.outputFormat == kubectlOutputFormatJSON || c.outputFormat == kubectlOutputFormatYaml {
			args = []string{
				"get",
				c.resource,
				"-o",
				c.outputFormat,
				name,
			}
		} else {
			panic(errorInvalidArgumentOutputFormat)
		}
		out, err = runKubectl(ctx, args)
		if err != nil {
			return 2, fmt.Errorf("failed to output: %w. Output command result: %s", err, string(out))
		}
	}

	if _, err := ioOut.Write(out); err != nil {
		return 1, err
	}
	return 0, nil
}

func buildCommand(name string, command string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(command)
	if err != nil {
		return "", fmt.Errorf("failed to parse preview command: %w", err)
	}
	builder := strings.Builder{}
	if err = tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to parse preview command: %w", err)
	}
	return builder.String(), nil
}
