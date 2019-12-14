package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"strings"
)

type getCommand struct {
	resource     string
	outputFormat string
	fzfOption    string
}

const (
	kubectlOutputFormatName     = "name"
	kubectlOutputFormatDescribe = "describe"
	kubectlOutputFormatYaml     = "yaml"
	kubectlOutputFormatJSON     = "json"

	previewCommandDescribe = "kubectl describe {{ .resource }} {{ .name }}"
	previewCommandYaml     = "kubectl get {{ .resource }} {{ .name }} -o yaml"

	envNameFzfOption     = "KUBECTL_FZF_FZF_OPTION"
	envNameFzfBindOption = "KUBECTL_FZF_FZF_BIND_OPTION"
	defaultFzfBindOption = "ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down"
	defaultFzfOption     = "--inline-info --layout reverse --preview '$KUBECTL_FZF_FZF_PREVIEW_OPTION' --preview-window down:70% --header-lines 1 --bind $KUBECTL_FZF_FZF_BIND_OPTION"
)

var (
	errorInvalidArgumentResource       = errors.New("1st argument must be the kind of kubernetes resources")
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

func NewGetCommand(resource string, previewFormat string, outputFormat string, fzfQuery string) (*getCommand, error) {
	if resource == "" {
		return nil, errorInvalidArgumentResource
	}
	previewCommandTemplate, ok := previewCommands[previewFormat]
	if !ok {
		return nil, errorInvalidArgumentPreviewCommand
	}
	previewCommand, err := buildCommand("preview", previewCommandTemplate, map[string]interface{}{
		"resource": resource,
		"name":     "{1}",
	})
	if err != nil {
		return nil, fmt.Errorf("invalid fzf preview command: %w", err)
	}

	fzfOption, err := getFzfOption(previewCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to get fzf option: %w", err)
	}
	if fzfQuery != "" {
		fzfOption = fzfOption + " --query " + fzfQuery
	}
	if _, ok := outputFormats[outputFormat]; !ok {
		return nil, errorInvalidArgumentOutputFormat
	}

	return &getCommand{
		resource:     resource,
		outputFormat: outputFormat,
		fzfOption:    fzfOption,
	}, nil
}

func getFzfOption(previewCommand string) (string, error) {
	fzfOption := os.Getenv(envNameFzfOption)
	if fzfOption == "" {
		fzfOption = defaultFzfOption
	}
	options := map[string][]string{
		"KUBECTL_FZF_FZF_PREVIEW_OPTION": {
			previewCommand,
		},
		envNameFzfBindOption: {
			os.Getenv(envNameFzfBindOption),
			defaultFzfBindOption,
		},
	}
	var invalidEnvVars []string
	fzfOption = os.Expand(fzfOption, func(envName string) string {
		for _, opt := range options[envName] {
			if opt != "" {
				return opt
			}
		}
		invalidEnvVars = append(invalidEnvVars, envName)
		return ""
	})
	if len(invalidEnvVars) != 0 {
		return "", fmt.Errorf("%s has invalid environment variables: %s", envNameFzfOption, strings.Join(invalidEnvVars, ","))
	}
	return fzfOption, nil
}

func (c getCommand) Run(ctx context.Context, ioIn io.Reader, ioOut io.Writer, ioErr io.Writer) error {
	kubectlCommand := fmt.Sprintf("kubectl get %s", c.resource)
	fzfCommandLine := fmt.Sprintf("fzf %s", c.fzfOption)
	commandLine := fmt.Sprintf("%s | %s", kubectlCommand, fzfCommandLine)

	out, err := runCommandWithFzf(ctx, commandLine, ioIn, ioErr)
	if err != nil {
		return fmt.Errorf("failed to run the command: %w", err)
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
			return fmt.Errorf("failed get kubernetes resource: %w. kubectl output: %s", err, string(out))
		}
	}

	if _, err := ioOut.Write(out); err != nil {
		return fmt.Errorf("failed to output the result: %w", err)
	}
	return nil
}

func buildCommand(name string, command string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(command)
	if err != nil {
		return "", fmt.Errorf("failed to parse the command: %w", err)
	}
	builder := strings.Builder{}
	if err = tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to set data on the template of command: %w", err)
	}
	return builder.String(), nil
}
