package command

import (
	"context"
	"errors"
	"fmt"
	"html/template"
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

	previewCommandDescribe = "kubectl describe {{ .resource }} {{ .name }}{{ .options }}"
	previewCommandYaml     = "kubectl get {{ .resource }} {{ .name }} -o yaml{{ .options }}"

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

func commandFromTemplate(name string, command string, data map[string]interface{}) (string, error) {
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
