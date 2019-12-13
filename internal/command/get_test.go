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
			outputFormat:   kubectlOutputFormatYaml,
			want:           &getCommand{resource: kubernetesResourcePods, previewCommand: previewCommandDescribe, outputFormat: kubectlOutputFormatYaml},
			wantErr:        nil,
		},
		{
			name:           "get yaml preview command",
			resource:       kubernetesResourceService,
			previewCommand: kubectlOutputFormatYaml,
			outputFormat:   kubectlOutputFormatYaml,
			want:           &getCommand{resource: kubernetesResourceService, previewCommand: previewCommandYaml, outputFormat: kubectlOutputFormatYaml},
			wantErr:        nil,
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := NewGetCommand(tc.resource, tc.previewCommand, tc.outputFormat)
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
