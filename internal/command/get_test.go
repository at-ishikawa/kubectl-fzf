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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kubernetesResourcePods    = "pods"
	kubernetesResourceService = "svc"
)

func TestNewGetCli(t *testing.T) {
	fzfOptionFunc := func(previewCommand string, hasMultipleResources bool, query string) string {
		fzf, err := getFzfOption(previewCommand, hasMultipleResources)
		if err != nil {
			panic(err)
		}
		if query != "" {
			return fzf + " --query " + query
		}
		return fzf
	}

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
			name:           "desc preview command for single resource",
			resource:       kubernetesResourcePods,
			namespace:      "default",
			previewCommand: kubectlOutputFormatDescribe,
			fzfQuery:       "",
			want: &getCli{
				kubectl: &kubectl{
					resource:  kubernetesResourcePods,
					namespace: "default",
				},
				fzfOption: fzfOptionFunc("kubectl describe pods {1} -n=default", false, ""),
			},
			wantErr: nil,
		},
		{
			name:           "desc preview command for all resources",
			resource:       kubernetesResourceAll,
			previewCommand: kubectlOutputFormatDescribe,
			want: &getCli{
				kubectl: &kubectl{
					resource: kubernetesResourceAll,
				},
				getOptions: map[string]string{
					"--no-headers": "true",
				},
				fzfOption: fzfOptionFunc("kubectl describe {1}", true, ""),
			},
			wantErr: nil,
		},
		{
			name:           "get yaml preview command for multiple resources",
			resource:       kubernetesResourcePods + "," + kubernetesResourceService,
			previewCommand: kubectlOutputFormatYaml,
			fzfQuery:       "svc",
			want: &getCli{
				kubectl: &kubectl{
					resource: kubernetesResourcePods + "," + kubernetesResourceService,
				},
				getOptions: map[string]string{
					"--no-headers": "true",
				},
				fzfOption: fzfOptionFunc("kubectl get {1} -o=yaml", true, "svc"),
			},
			wantErr: nil,
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
			name:           "KUBECTL_FZF_FZF_OPTION includes invalid env",
			resource:       kubernetesResourcePods,
			previewCommand: kubectlOutputFormatYaml,
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
			k := &kubectl{
				resource:  tc.resource,
				namespace: tc.namespace,
			}
			got, gotErr := NewGetCli(k, tc.previewCommand, tc.fzfQuery)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestGetCli_Run(t *testing.T) {
	fzfOption := "--inline-info"
	defaultRunCommand := func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
		assert.Contains(t, commandLine, fmt.Sprintf("| fzf %s", fzfOption))
		return bytes.NewBufferString("pod 2/2 Running 2d").Bytes(), nil
	}
	defaultWantErr := errors.New("want error")
	exitErr := exec.ExitError{}

	testCases := []struct {
		name                string
		runCommandWithFzf   func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error)
		sut                 getCli
		kubectlGetErr       error
		kubectlGetDetailErr error
		wantErr             error
		wantIO              string
		wantIOErr           string
	}{
		{
			name: "normal case",
			sut: getCli{
				kubectl: &kubectl{
					resource: kubernetesResourcePods,
				},
				fzfOption: fzfOption,
			},
			runCommandWithFzf: defaultRunCommand,
			wantErr:           nil,
			wantIO:            "pod\n",
			wantIOErr:         "",
		},
		{
			name: "command with fzf error",
			sut: getCli{
				kubectl: &kubectl{
					resource: kubernetesResourcePods,
				},
				fzfOption: fzfOption,
			},
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, defaultWantErr
			},
			wantErr:   defaultWantErr,
			wantIO:    "",
			wantIOErr: "",
		},
		{
			name: "invalid namespace",
			sut: getCli{
				kubectl: &kubectl{
					resource:  kubernetesResourcePods,
					namespace: "invalid",
				},
				fzfOption: fzfOption,
			},
			runCommandWithFzf: defaultRunCommand,
			kubectlGetErr:     &exitErr,
			wantErr:           &exitErr,
			wantIO:            "",
			wantIOErr:         "",
		},
		{
			name: "command with fzf exit error (not 130)",
			sut: getCli{
				kubectl: &kubectl{
					resource: kubernetesResourcePods,
				},
				fzfOption: fzfOption,
			},
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, &exitErr
			},
			wantErr:   &exitErr,
			wantIO:    "",
			wantIOErr: "",
		},
		{
			name: "kubectl get command error",
			sut: getCli{
				kubectl: &kubectl{
					resource: kubernetesResourcePods,
				},
				fzfOption: fzfOption,
			},
			runCommandWithFzf: defaultRunCommand,
			kubectlGetErr:     defaultWantErr,
			wantErr:           defaultWantErr,
			wantIO:            "",
			wantIOErr:         "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubectl := NewMockKubectl(mockCtrl)
			mockKubectl.EXPECT().
				run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]byte("Name Ready Status Age\npod 2/2 Running 2d"), tc.kubectlGetErr).
				Times(1)
			mockKubectl.EXPECT().
				run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]byte("apiVersion: v1\nkind: Pod"), tc.kubectlGetDetailErr).
				MaxTimes(1)
			runCommandWithFzf = tc.runCommandWithFzf
			tc.sut.kubectl = mockKubectl

			var gotIOOut bytes.Buffer
			var gotIOErr bytes.Buffer
			gotErr := tc.sut.Run(context.Background(), strings.NewReader("in"), &gotIOOut, &gotIOErr)
			assert.True(t, errors.Is(gotErr, tc.wantErr))
			assert.Equal(t, tc.wantIO, gotIOOut.String())
			assert.Equal(t, tc.wantIOErr, gotIOErr.String())
		})
	}
}
