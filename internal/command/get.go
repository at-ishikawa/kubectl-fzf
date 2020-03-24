package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"io"
	"os/exec"
	"strings"
)

type getCli struct {
	resource     string
	outputFormat string
	fzfOption    string
}

var (
	errorInvalidArgumentFZFPreviewCommand = errors.New("preview format must be one of [describe, yaml]")
	errorInvalidArgumentOutputFormat      = errors.New("output format must be one of [name, yaml, json]")

	getCliPreviewCommands = map[string]string{
		kubectlOutputFormatDescribe: previewCommandDescribe,
		kubectlOutputFormatYaml:     previewCommandYaml,
	}
	getCliOutputFormats = map[string]struct{}{
		kubectlOutputFormatName: {},
		kubectlOutputFormatYaml: {},
		kubectlOutputFormatJSON: {},
	}
)

func NewGetCli(kubernetesResource string, previewFormat string, outputFormat string, fzfQuery string) (*getCli, error) {
	if kubernetesResource == "" {
		return nil, errorInvalidArgumentKubernetesResource
	}
	previewCommandTemplate, ok := getCliPreviewCommands[previewFormat]
	if !ok {
		return nil, errorInvalidArgumentFZFPreviewCommand
	}
	previewCommand, err := commandFromTemplate("preview", previewCommandTemplate, map[string]interface{}{
		"resource": kubernetesResource,
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
	if _, ok := getCliOutputFormats[outputFormat]; !ok {
		return nil, errorInvalidArgumentOutputFormat
	}

	return &getCli{
		resource:     kubernetesResource,
		fzfOption:    fzfOption,
		outputFormat: outputFormat,
	}, nil
}

func (c getCli) Run(ctx context.Context, ioIn io.Reader, ioOut io.Writer, ioErr io.Writer) error {
	command := fmt.Sprintf("kubectl get %s | fzf %s", c.resource, c.fzfOption)
	out, err := runCommandWithFzf(ctx, command, ioIn, ioErr)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Script canceled by Ctrl-c
			// Only for bash?: http://tldp.org/LDP/abs/html/exitcodes.html
			if exitErr.ExitCode() == 130 {
				return nil
			}
		}
		return fmt.Errorf("failed to run the command %s: %w", command, err)
	}

	line := strings.TrimSpace(string(out))
	columns := strings.Fields(line)
	name := strings.TrimSpace(columns[0])

	if c.outputFormat == kubectlOutputFormatName {
		out = bytes.NewBufferString(name + "\n").Bytes()
	} else {
		var args []string
		if c.outputFormat == kubectlOutputFormatJSON || c.outputFormat == kubectlOutputFormatYaml {
			args = []string{
				"get",
				c.resource,
				"-o",
				c.outputFormat,
				name,
			}
		} else {
			// The output format has to be validated on NewGetCli function
			// So this should never happens
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
