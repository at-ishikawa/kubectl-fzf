package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kubernetesResourcePods    = "pods"
	kubernetesResourceService = "svc"
)

func TestMain(m *testing.M) {
	backupRunKubectlFunc := runKubectl
	backupRunCommandWithFzf := runCommandWithFzf
	defer func() {
		runKubectl = backupRunKubectlFunc
		runCommandWithFzf = backupRunCommandWithFzf
	}()
	os.Exit(m.Run())
}

func TestGetFzfOption(t *testing.T) {
	testCases := []struct {
		name           string
		previewCommand string
		envVars        map[string]string
		want           string
		wantErr        error
	}{
		{
			name:           "no env vars",
			previewCommand: "kubectl describe pods {{1}}",
			want:           fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s", "kubectl describe pods {{1}}", defaultFzfBindOption),
		},
		{
			name:           "all correct env vars",
			previewCommand: "kubectl describe pods {{1}}",
			envVars: map[string]string{
				envNameFzfOption:     fmt.Sprintf("--preview '$KUBECTL_FZF_FZF_PREVIEW_OPTION' --bind $%s", envNameFzfBindOption),
				envNameFzfBindOption: "ctrl-k:kill-line",
			},
			want: fmt.Sprintf("--preview '%s' --bind %s", "kubectl describe pods {{1}}", "ctrl-k:kill-line"),
		},
		{
			name:           "no env vars",
			previewCommand: "unused preview command",
			envVars: map[string]string{
				envNameFzfOption:     "--inline-info",
				envNameFzfBindOption: "unused",
			},
			want: "--inline-info",
		},
		{
			name:           "invalid env vars in KUBECTL_FZF_FZF_OPTION",
			previewCommand: "unused preview command",
			envVars: map[string]string{
				envNameFzfOption:     "--inline-info $UNKNOWN_ENV_NAME",
				envNameFzfBindOption: "unused",
			},
			want:    "",
			wantErr: fmt.Errorf("%s has invalid environment variables: UNKNOWN_ENV_NAME", envNameFzfOption),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				for k := range tc.envVars {
					require.NoError(t, os.Unsetenv(k))
				}
			}()
			for k, v := range tc.envVars {
				require.NoError(t, os.Setenv(k, v))
			}
			got, gotErr := getFzfOption(tc.previewCommand)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestNewGetCommand(t *testing.T) {
	testCases := []struct {
		name           string
		resource       string
		previewCommand string
		outputFormat   string
		envVars        map[string]string
		want           *getCommand
		wantErr        error
	}{
		{
			name:           "desc preview command",
			resource:       kubernetesResourcePods,
			previewCommand: kubectlOutputFormatDescribe,
			outputFormat:   kubectlOutputFormatYaml,
			want: &getCommand{
				resource:     kubernetesResourcePods,
				outputFormat: kubectlOutputFormatYaml,
				fzfOption:    fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s", "kubectl describe pods {1}", defaultFzfBindOption),
			},
			wantErr: nil,
		},
		{
			name:           "get yaml preview command",
			resource:       kubernetesResourceService,
			previewCommand: kubectlOutputFormatYaml,
			outputFormat:   kubectlOutputFormatYaml,
			want: &getCommand{
				resource:     kubernetesResourceService,
				outputFormat: kubectlOutputFormatYaml,
				fzfOption:    fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s", "kubectl get svc {1} -o yaml", defaultFzfBindOption),
			},
			wantErr: nil,
		},
		{
			name:           "empty yaml",
			resource:       "",
			previewCommand: kubectlOutputFormatYaml,
			outputFormat:   kubectlOutputFormatYaml,
			want:           nil,
			wantErr:        errorInvalidArgumentResource,
		},
		{
			name:           "invalid preview command",
			resource:       kubernetesResourcePods,
			previewCommand: "unknown",
			outputFormat:   kubectlOutputFormatYaml,
			want:           nil,
			wantErr:        errorInvalidArgumentPreviewCommand,
		},
		{
			name:           "empty preview command",
			resource:       kubernetesResourcePods,
			previewCommand: "",
			want:           nil,
			wantErr:        errorInvalidArgumentPreviewCommand,
		},
		{
			name:           "invalid output format",
			resource:       kubernetesResourcePods,
			previewCommand: kubectlOutputFormatYaml,
			outputFormat:   "unknown",
			want:           nil,
			wantErr:        errorInvalidArgumentOutputFormat,
		},
		{
			name:           "empty output format",
			resource:       kubernetesResourcePods,
			previewCommand: kubectlOutputFormatYaml,
			outputFormat:   "",
			want:           nil,
			wantErr:        errorInvalidArgumentOutputFormat,
		},
		{
			name:           "KUBECTL_FZF_FZF_OPTION includes invalid env",
			resource:       kubernetesResourcePods,
			previewCommand: kubectlOutputFormatYaml,
			outputFormat:   kubectlOutputFormatYaml,
			envVars: map[string]string{
				envNameFzfOption: "$UNKNOWN_ENV1, $UNKNOWN_ENV2",
			},
			want:    nil,
			wantErr: fmt.Errorf("failed to get fzf option: %w", fmt.Errorf("%s has invalid environment variables: %s", envNameFzfOption, "UNKNOWN_ENV1,UNKNOWN_ENV2")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.envVars) > 0 {
				defer func() {
					for k := range tc.envVars {
						require.NoError(t, os.Unsetenv(k))
					}
				}()
				for k, v := range tc.envVars {
					require.NoError(t, os.Setenv(k, v))
				}
			}
			got, gotErr := NewGetCommand(tc.resource, tc.previewCommand, tc.outputFormat)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestRun(t *testing.T) {
	fzfOption := "--inline-info"
	defaultRunCommand := func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
		assert.Equal(t, fmt.Sprintf("%s | fzf %s",
			"kubectl get pods",
			fzfOption,
		), commandLine)
		return bytes.NewBufferString("pod 2/2 Running 2d").Bytes(), nil
	}
	defaultWantErr := errors.New("want error")

	testCases := []struct {
		name              string
		runCommandWithFzf func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error)
		runKubectl        func(ctx context.Context, args []string) (i []byte, e error)
		sut               getCommand
		wantErr           error
		wantIO            string
		wantIOErr         string
	}{
		{
			name: "name output",
			sut: getCommand{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatName,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl:        nil,
			wantErr:           nil,
			wantIO:            "pod",
			wantIOErr:         "",
		},
		{
			name: "yaml output",
			sut: getCommand{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatYaml,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl: func(ctx context.Context, args []string) (i []byte, e error) {
				assert.Equal(t, []string{
					"get",
					kubernetesResourcePods,
					"-o",
					kubectlOutputFormatYaml,
					"pod",
				}, args)
				return bytes.NewBufferString("Kind: Pod").Bytes(), nil
			},
			wantErr:   nil,
			wantIO:    "Kind: Pod",
			wantIOErr: "",
		},
		{
			name: "json output",
			sut: getCommand{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatJSON,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl: func(ctx context.Context, args []string) (i []byte, e error) {
				assert.Equal(t, []string{
					"get",
					kubernetesResourcePods,
					"-o",
					kubectlOutputFormatJSON,
					"pod",
				}, args)
				return bytes.NewBufferString("{\"kind\":\"Pod\"}").Bytes(), nil
			},
			wantErr:   nil,
			wantIO:    "{\"kind\":\"Pod\"}",
			wantIOErr: "",
		},
		{
			name: "describe output",
			sut: getCommand{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatDescribe,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl: func(ctx context.Context, args []string) (i []byte, e error) {
				assert.Equal(t, []string{
					"describe",
					kubernetesResourcePods,
					"pod",
				}, args)
				return bytes.NewBufferString("Name: pod").Bytes(), nil
			},
			wantErr:   nil,
			wantIO:    "Name: pod",
			wantIOErr: "",
		},
		{
			name: "command with fzf error",
			sut: getCommand{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatDescribe,
			},
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, defaultWantErr
			},
			runKubectl: nil,
			wantErr:    defaultWantErr,
			wantIO:     "",
			wantIOErr:  "",
		},
		{
			name: "kubectl command error",
			sut: getCommand{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatDescribe,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl: func(ctx context.Context, args []string) (i []byte, e error) {
				return nil, defaultWantErr
			},
			wantErr:   defaultWantErr,
			wantIO:    "",
			wantIOErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runCommandWithFzf = tc.runCommandWithFzf
			runKubectl = tc.runKubectl

			var gotIOOut bytes.Buffer
			var gotIOErr bytes.Buffer
			gotErr := tc.sut.Run(context.Background(), strings.NewReader("in"), &gotIOOut, &gotIOErr)
			assert.True(t, errors.Is(gotErr, tc.wantErr))
			assert.Equal(t, tc.wantIO, gotIOOut.String())
			assert.Equal(t, tc.wantIOErr, gotIOErr.String())
		})
	}
}

func TestBuildCommand(t *testing.T) {
	testCases := []struct {
		name         string
		templateName string
		command      string
		data         map[string]interface{}
		want         string
		wantIsErr    bool
	}{
		{
			name:         "template",
			templateName: "template",
			command:      "kubectl {{ .command }} {{ .resource }}",
			data: map[string]interface{}{
				"command":  "get",
				"resource": "pods",
			},
			want:      "kubectl get pods",
			wantIsErr: false,
		},
		{
			name:         "no template",
			templateName: "",
			command:      "{{ .name }}",
			data: map[string]interface{}{
				"name": "fzf",
			},
			want:      "fzf",
			wantIsErr: false,
		},
		{
			name:         "invalid command",
			templateName: "template",
			command:      "{{ .name }",
			data: map[string]interface{}{
				"name": "name",
			},
			want:      "",
			wantIsErr: true,
		},
		{
			name:         "wrong parameter",
			templateName: "template",
			command:      "wrong {{ .name }}",
			data: map[string]interface{}{
				"unknown": "unknown",
			},
			want:      "wrong ",
			wantIsErr: false,
		},
		{
			name:         "no parameter",
			templateName: "template",
			command:      "no {{ .name }}",
			data:         nil,
			want:         "no ",
			wantIsErr:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := buildCommand(tc.templateName, tc.command, tc.data)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantIsErr, gotErr != nil)
		})
	}
}
