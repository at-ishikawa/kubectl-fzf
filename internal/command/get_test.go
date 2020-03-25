package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kubernetesResourcePods    = "pods"
	kubernetesResourceService = "svc"
)

func TestNewGetCli(t *testing.T) {
	testCases := []struct {
		name           string
		resource       string
		namespace      string
		previewCommand string
		outputFormat   string
		fzfQuery       string
		envVars        map[string]string
		want           *getCli
		wantErr        error
	}{
		{
			name:           "desc preview command",
			resource:       kubernetesResourcePods,
			namespace:      "default",
			previewCommand: kubectlOutputFormatDescribe,
			outputFormat:   kubectlOutputFormatYaml,
			fzfQuery:       "",
			want: &getCli{
				resource:     kubernetesResourcePods,
				namespace:    "default",
				outputFormat: kubectlOutputFormatYaml,
				fzfOption:    fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s", "kubectl describe pods {1} -n default", defaultFzfBindOption),
			},
			wantErr: nil,
		},
		{
			name:           "get yaml preview command",
			resource:       kubernetesResourceService,
			previewCommand: kubectlOutputFormatYaml,
			outputFormat:   kubectlOutputFormatYaml,
			fzfQuery:       "svc",
			want: &getCli{
				resource:     kubernetesResourceService,
				outputFormat: kubectlOutputFormatYaml,
				fzfOption:    fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s --query svc", "kubectl get svc {1} -o yaml", defaultFzfBindOption),
			},
			wantErr: nil,
		},
		{
			name:           "empty yaml",
			resource:       "",
			previewCommand: kubectlOutputFormatYaml,
			outputFormat:   kubectlOutputFormatYaml,
			want:           nil,
			wantErr:        errorInvalidArgumentKubernetesResource,
		},
		{
			name:           "invalid preview command",
			resource:       kubernetesResourcePods,
			previewCommand: "unknown",
			outputFormat:   kubectlOutputFormatYaml,
			want:           nil,
			wantErr:        errorInvalidArgumentFZFPreviewCommand,
		},
		{
			name:           "empty preview command",
			resource:       kubernetesResourcePods,
			previewCommand: "",
			want:           nil,
			wantErr:        errorInvalidArgumentFZFPreviewCommand,
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
			got, gotErr := NewGetCli(tc.resource, tc.namespace, tc.previewCommand, tc.outputFormat, tc.fzfQuery)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestGetCli_Run(t *testing.T) {
	fzfOption := "--inline-info"
	defaultGetCommand := func(_ context.Context, _ []string) ([]byte, error) {
		return bytes.NewBufferString("Name\npod").Bytes(), nil
	}
	defaultRunCommand := func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
		assert.Equal(t, fmt.Sprintf("echo 'Name\npod' | fzf %s",
			fzfOption,
		), commandLine)
		return bytes.NewBufferString("pod 2/2 Running 2d").Bytes(), nil
	}
	defaultWantErr := errors.New("want error")
	exitErr := exec.ExitError{}

	testCases := []struct {
		name              string
		runCommandWithFzf func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error)
		runKubectl        func(ctx context.Context, args []string) (i []byte, e error)
		sut               getCli
		wantErr           error
		wantIO            string
		wantIOErr         string
	}{
		{
			name: "name output",
			sut: getCli{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatName,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl:        defaultGetCommand,
			wantErr:           nil,
			wantIO:            "pod\n",
			wantIOErr:         "",
		},
		{
			name: "yaml output",
			sut: getCli{
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
			sut: getCli{
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
			name: "command with fzf error",
			sut: getCli{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatYaml,
			},
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, defaultWantErr
			},
			runKubectl: defaultGetCommand,
			wantErr:    defaultWantErr,
			wantIO:     "",
			wantIOErr:  "",
		},
		{
			name: "invalid namespace",
			sut: getCli{
				resource:     kubernetesResourcePods,
				namespace:    "invalid",
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatYaml,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl: func(ctx context.Context, args []string) (i []byte, e error) {
				return bytes.NewBufferString("No resources found").Bytes(), nil
			},
			wantErr:   &exitErr,
			wantIO:    "",
			wantIOErr: "",
		},
		{
			name: "command with fzf exit error (not 130)",
			sut: getCli{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatYaml,
			},
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, &exitErr
			},
			runKubectl: defaultGetCommand,
			wantErr:    &exitErr,
			wantIO:     "",
			wantIOErr:  "",
		},
		{
			name: "kubectl command error",
			sut: getCli{
				resource:     kubernetesResourcePods,
				fzfOption:    fzfOption,
				outputFormat: kubectlOutputFormatYaml,
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
