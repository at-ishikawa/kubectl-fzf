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
	kubectl      Kubectl
	outputFormat string
	fzfOption    string
}

var (
	errorInvalidArgumentFZFPreviewCommand = errors.New("preview format must be one of [describe, yaml]")
	errorInvalidArgumentOutputFormat      = errors.New("output format must be one of [name, yaml, json]")

	getCliPreviewCommands = map[string]struct {
		operation string
		options   map[string]string
	}{
		kubectlOutputFormatDescribe: {
			operation: "describe",
		},
		kubectlOutputFormatYaml: {
			operation: "get",
			options: map[string]string{
				"-o": "yaml",
			},
		},
	}
	getCliOutputFormats = map[string]struct{}{
		kubectlOutputFormatName: {},
		kubectlOutputFormatYaml: {},
		kubectlOutputFormatJSON: {},
	}
)

func NewGetCli(k *kubectl, previewFormat string, outputFormat string, fzfQuery string) (*getCli, error) {
	previewCommandTemplate, ok := getCliPreviewCommands[previewFormat]
	if !ok {
		return nil, errorInvalidArgumentFZFPreviewCommand
	}
	previewCommand := k.getCommand(previewCommandTemplate.operation, "{1}", previewCommandTemplate.options)
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
		kubectl:      k,
		fzfOption:    fzfOption,
		outputFormat: outputFormat,
	}, nil
}

func (c getCli) Run(ctx context.Context, ioIn io.Reader, ioOut io.Writer, ioErr io.Writer) error {
	out, err := c.kubectl.run(ctx, "get", "", nil)
	if err != nil {
		return err
	}
	if len(strings.Split(strings.TrimSpace(string(out)), "\n")) == 1 {
		return fmt.Errorf("failed to run kubectl. Namespace may not exist")
	}
	command := fmt.Sprintf("echo '%s' | fzf %s", string(out), c.fzfOption)
	out, err = runCommandWithFzf(ctx, command, ioIn, ioErr)
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
		out, err = c.kubectl.run(ctx, "get", name, map[string]string{
			"-o": c.outputFormat,
		})
		if err != nil {
			return err
		}
	}

	if _, err := ioOut.Write(out); err != nil {
		return fmt.Errorf("failed to output the result: %w", err)
	}
	return nil
}
