package command

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type describeCli struct {
	kubectl   Kubectl
	fzfOption string
}

func NewDescribeCli(kubectl Kubectl, fzfQuery string) (*describeCli, error) {
	previewCommand := kubectl.getCommand("describe", "{1}", nil)
	fzfOption, err := getFzfOption(previewCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to get fzf option: %w", err)
	}
	if fzfQuery != "" {
		fzfOption = fzfOption + " --query " + fzfQuery
	}

	return &describeCli{
		kubectl:   kubectl,
		fzfOption: fzfOption,
	}, nil
}

func (c describeCli) Run(ctx context.Context, ioIn io.Reader, ioOut io.Writer, ioErr io.Writer) error {
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

	out, err = c.kubectl.run(ctx, "describe", name, nil)
	if err != nil {
		return err
	}
	if _, err := ioOut.Write(out); err != nil {
		return fmt.Errorf("failed to output the result: %w", err)
	}
	return nil
}
