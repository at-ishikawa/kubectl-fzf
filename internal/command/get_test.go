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

func TestNewGetCommand(t *testing.T) {
	testCases := []struct {
		name           string
		resource       string
		previewCommand string
		outputFormat   string
		want           *getCommand
		wantErr        error
	}{
		{
			name:           "desc preview command",
			resource:       kubernetesResourcePods,
			previewCommand: kubectlOutputFormatDescribe,
			want:           &getCommand{resource: kubernetesResourcePods, previewCommand: previewCommandDescribe, outputFormat: kubectlOutputFormatYaml},
			wantErr:        nil,
		},
		{
			name:           "get yaml preview command",
			resource:       kubernetesResourceService,
			previewCommand: kubectlOutputFormatYaml,
			want:           &getCommand{resource: kubernetesResourceService, previewCommand: previewCommandYaml, outputFormat: kubectlOutputFormatYaml},
			wantErr:        nil,
		},
		{
			name:           "empty yaml",
			resource:       "",
			previewCommand: kubectlOutputFormatYaml,
			want:           nil,
			wantErr:        errorInvalidArgumentResource,
		},
		{
			name:           "invalid preview command",
			resource:       kubernetesResourcePods,
			previewCommand: "unknown",
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := NewGetCommand(tc.resource, tc.previewCommand, kubectlOutputFormatYaml)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestRun(t *testing.T) {
	defaultRunCommand := func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
		assert.Equal(t, fmt.Sprintf("%s | %s",
			"kubectl get pods",
			"fzf --inline-info --layout reverse --preview 'preview' --preview-window down:70% --header-lines 1 --bind ctrl-k:kill-line,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down",
		), commandLine)
		return bytes.NewBufferString("pod 2/2 Running 2d").Bytes(), nil
	}
	defaultWantErr := errors.New("want error")

	testCases := []struct {
		name              string
		runCommandWithFzf func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error)
		runKubectl        func(ctx context.Context, args []string) (i []byte, e error)
		sut               getCommand
		want              int
		wantErr           error
		wantIO            string
		wantIOErr         string
	}{
		{
			name: "name output",
			sut: getCommand{
				resource:       kubernetesResourcePods,
				previewCommand: "preview",
				outputFormat:   kubectlOutputFormatName,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl:        nil,
			want:              0,
			wantErr:           nil,
			wantIO:            "pod",
			wantIOErr:         "",
		},
		{
			name: "yaml output",
			sut: getCommand{
				resource:       kubernetesResourcePods,
				previewCommand: "preview",
				outputFormat:   kubectlOutputFormatYaml,
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
			want:      0,
			wantErr:   nil,
			wantIO:    "Kind: Pod",
			wantIOErr: "",
		},
		{
			name: "json output",
			sut: getCommand{
				resource:       kubernetesResourcePods,
				previewCommand: "preview",
				outputFormat:   kubectlOutputFormatJSON,
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
			want:      0,
			wantErr:   nil,
			wantIO:    "{\"kind\":\"Pod\"}",
			wantIOErr: "",
		},
		{
			name: "describe output",
			sut: getCommand{
				resource:       kubernetesResourcePods,
				previewCommand: "preview",
				outputFormat:   kubectlOutputFormatDescribe,
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
			want:      0,
			wantErr:   nil,
			wantIO:    "Name: pod",
			wantIOErr: "",
		},
		{
			name: "command with fzf error",
			sut: getCommand{
				resource:       kubernetesResourcePods,
				previewCommand: "preview",
				outputFormat:   kubectlOutputFormatDescribe,
			},
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, defaultWantErr
			},
			runKubectl: nil,
			want:       1,
			wantErr:    defaultWantErr,
			wantIO:     "",
			wantIOErr:  "",
		},
		{
			name: "kubectl command error",
			sut: getCommand{
				resource:       kubernetesResourcePods,
				previewCommand: "preview",
				outputFormat:   kubectlOutputFormatDescribe,
			},
			runCommandWithFzf: defaultRunCommand,
			runKubectl: func(ctx context.Context, args []string) (i []byte, e error) {
				return nil, defaultWantErr
			},
			want:      2,
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
			gotExitCode, gotErr := tc.sut.Run(context.Background(), strings.NewReader("in"), &gotIOOut, &gotIOErr)
			assert.Equal(t, tc.want, gotExitCode)
			assert.True(t, errors.Is(gotErr, tc.wantErr))
			assert.Equal(t, tc.wantIO, gotIOOut.String())
			assert.Equal(t, tc.wantIOErr, gotIOErr.String())
		})
	}
}
