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
	kubectl    Kubectl
	getOptions map[string]string
	fzfOption  string
}

var (
	errorInvalidArgumentFZFPreviewCommand = errors.New("preview format must be one of [describe, yaml]")

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
)

func NewGetCli(k *kubectl, previewFormat string, fzfQuery string) (*getCli, error) {
	previewCommandTemplate, ok := getCliPreviewCommands[previewFormat]
	if !ok {
		return nil, errorInvalidArgumentFZFPreviewCommand
	}

	resource := k.resource
	var getOptions map[string]string
	hasMultipleResources := false
	if k.resource == kubernetesResourceAll || strings.Contains(k.resource, ",") {
		resource = ""
		getOptions = map[string]string{
			"--no-headers": "true",
		}
		hasMultipleResources = true
	}
	previewCommand := k.getCommand(previewCommandTemplate.operation, resource, []string{"{1}"}, previewCommandTemplate.options)
	fzfOption, err := getFzfOption(previewCommand, hasMultipleResources)
	if err != nil {
		return nil, fmt.Errorf("failed to get fzf option: %w", err)
	}
	if fzfQuery != "" {
		fzfOption = fzfOption + " --query " + fzfQuery
	}

	return &getCli{
		kubectl:    k,
		getOptions: getOptions,
		fzfOption:  fzfOption,
	}, nil
}

func (c getCli) Run(ctx context.Context, ioIn io.Reader, ioOut io.Writer, ioErr io.Writer) error {
	out, err := c.kubectl.run(ctx, "get", nil, c.getOptions)
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

	rows := strings.Split(strings.TrimSpace(string(out)), "\n")
	names := make([]string, len(rows))
	for i, row := range rows {
		columns := strings.Fields(row)
		names[i] = strings.TrimSpace(columns[0])
	}

	out = bytes.NewBufferString(strings.Join(names, "\n") + "\n").Bytes()
	if _, err := ioOut.Write(out); err != nil {
		return fmt.Errorf("failed to output the result: %w", err)
	}
	return nil
}
