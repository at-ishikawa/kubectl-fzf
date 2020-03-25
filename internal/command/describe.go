package command

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type describeCli struct {
	resource  string
	namespace string
	fzfOption string
}

func NewDescribeCli(kubernetesResource string, kubernetesNamespace string, fzfQuery string) (*describeCli, error) {
	if kubernetesResource == "" {
		return nil, errorInvalidArgumentKubernetesResource
	}
	var options []string
	if kubernetesNamespace != "" {
		options = append(options, "-n", kubernetesNamespace)
	}
	option := ""
	if len(options) > 0 {
		option = " " + strings.Join(options, " ")
	}
	previewCommand, err := commandFromTemplate("preview", previewCommandDescribe, map[string]interface{}{
		"resource": kubernetesResource,
		"name":     "{1}",
		"options":  option,
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

	return &describeCli{
		resource:  kubernetesResource,
		namespace: kubernetesNamespace,
		fzfOption: fzfOption,
	}, nil
}

func (c describeCli) Run(ctx context.Context, ioIn io.Reader, ioOut io.Writer, ioErr io.Writer) error {
	arguments := []string{
		"get",
		c.resource,
	}
	var kubectlOptions []string
	if c.namespace != "" {
		kubectlOptions = append(kubectlOptions, "-n", c.namespace)
	}
	arguments = append(arguments, kubectlOptions...)
	out, err := runKubectl(ctx, arguments)
	if err != nil {
		return fmt.Errorf("failed to run kubectl: %w", err)
	}
	if len(strings.Split(strings.TrimSpace(string(out)), "\n")) == 1 {
		return fmt.Errorf("failed to run kubectl. Namespace %s may not exist", c.namespace)
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

	args := []string{
		"describe",
		c.resource,
		name,
	}
	args = append(args, kubectlOptions...)
	out, err = runKubectl(ctx, args)
	if err != nil {
		return fmt.Errorf("failed get kubernetes resource: %w. kubectl output: %s", err, string(out))
	}

	if _, err := ioOut.Write(out); err != nil {
		return fmt.Errorf("failed to output the result: %w", err)
	}
	return nil
}
