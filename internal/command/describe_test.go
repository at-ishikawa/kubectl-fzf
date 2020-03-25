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

func TestNewDescribeCli(t *testing.T) {
	testCases := []struct {
		name      string
		resource  string
		namespace string
		fzfQuery  string
		envVars   map[string]string
		want      *describeCli
		wantErr   error
	}{
		{
			name:      "desc preview command",
			resource:  kubernetesResourcePods,
			namespace: "default",
			fzfQuery:  "",
			want: &describeCli{
				resource:  kubernetesResourcePods,
				namespace: "default",
				fzfOption: fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s", "kubectl describe pods {1} -n default", defaultFzfBindOption),
			},
			wantErr: nil,
		},
		{
			name:     "no namespace",
			resource: kubernetesResourceService,
			fzfQuery: "",
			want: &describeCli{
				resource:  kubernetesResourceService,
				fzfOption: fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s", "kubectl describe svc {1}", defaultFzfBindOption),
			},
			wantErr: nil,
		},
		{
			name:     "empty resource",
			resource: "",
			want:     nil,
			wantErr:  errorInvalidArgumentKubernetesResource,
		},

		{
			name:     "KUBECTL_FZF_FZF_OPTION includes invalid env",
			resource: kubernetesResourcePods,
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
			got, gotErr := NewDescribeCli(tc.resource, tc.namespace, tc.fzfQuery)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestDescribeCli_Run(t *testing.T) {
	fzfOption := "--inline-info"
	defaultRunCommand := func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
		assert.Equal(t, fmt.Sprintf("%s | fzf %s",
			"kubectl get pods",
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
		sut               describeCli
		wantErr           error
		wantIO            string
		wantIOErr         string
	}{
		{
			name: "output",
			sut: describeCli{
				resource:  kubernetesResourcePods,
				fzfOption: fzfOption,
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
			sut: describeCli{
				resource:  kubernetesResourcePods,
				fzfOption: fzfOption,
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
			name: "command with fzf exit error (not 130)",
			sut: describeCli{
				resource:  kubernetesResourcePods,
				fzfOption: fzfOption,
			},
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, &exitErr
			},
			runKubectl: nil,
			wantErr:    &exitErr,
			wantIO:     "",
			wantIOErr:  "",
		},
		{
			name: "kubectl command error",
			sut: describeCli{
				resource:  kubernetesResourcePods,
				fzfOption: fzfOption,
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
