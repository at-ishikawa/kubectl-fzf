package command

//go:generate mockgen -destination=./kubectl_mock.go -package=command github.com/at-ishikawa/kubectl-fzf/internal/command Kubectl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	kubectlOutputFormatName     = "name"
	kubectlOutputFormatDescribe = "describe"
	kubectlOutputFormatYaml     = "yaml"
	kubectlOutputFormatJSON     = "json"

	envNameFzfOption     = "KUBECTL_FZF_FZF_OPTION"
	envNameFzfBindOption = "KUBECTL_FZF_FZF_BIND_OPTION"
	defaultFzfBindOption = "ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down"
	defaultFzfOption     = "--inline-info --layout reverse --preview '$KUBECTL_FZF_FZF_PREVIEW_OPTION' --preview-window down:70% --header-lines 1 --bind $KUBECTL_FZF_FZF_BIND_OPTION"
)

var (
	errorInvalidArgumentKubernetesResource = errors.New("1st argument must be the kind of kubernetes resources")

	runCommandWithFzf = func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "sh", "-c", commandLine)
		cmd.Stderr = ioErr
		cmd.Stdin = ioIn
		return cmd.Output()
	}
	runKubectl = func(ctx context.Context, args []string) ([]byte, error) {
		return exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
	}
)

type Kubectl interface {
	getCommand(operation string, name string, options map[string]string) string
	run(ctx context.Context, operation string, name string, options map[string]string) ([]byte, error)
}

type kubectl struct {
	resource  string
	namespace string
}

func NewKubectl(kubernetesResource string, kubernetesNamespace string) (*kubectl, error) {
	if kubernetesResource == "" {
		return nil, errorInvalidArgumentKubernetesResource
	}
	return &kubectl{
		resource:  kubernetesResource,
		namespace: kubernetesNamespace,
	}, nil
}

func (k kubectl) run(ctx context.Context, operation string, name string, options map[string]string) ([]byte, error) {
	out, err := runKubectl(ctx, k.getArguments(operation, name, options))
	if err != nil {
		return nil, fmt.Errorf("failed get kubernetes resource: %w. kubectl output: %s", err, string(out))
	}
	return out, nil
}

func (k kubectl) getCommand(operation string, name string, options map[string]string) string {
	return "kubectl " + strings.Join(k.getArguments(operation, name, options), " ")
}

func (k kubectl) getArguments(operation string, name string, options map[string]string) []string {
	args := []string{
		operation,
		k.resource,
	}
	if name != "" {
		args = append(args, name)
	}
	if k.namespace != "" {
		args = append(args, "-n", k.namespace)
	}
	for k, v := range options {
		args = append(args, k, v)
	}
	return args
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
