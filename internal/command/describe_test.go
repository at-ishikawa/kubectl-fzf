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

func TestNewDescribeCli(t *testing.T) {
	kubectlWithNamespace := kubectl{
		resource:  kubernetesResourcePods,
		namespace: "default",
	}
	kubectlWithoutNamespace := kubectl{
		resource: kubernetesResourceService,
	}

	testCases := []struct {
		name     string
		kubectl  kubectl
		fzfQuery string
		envVars  map[string]string
		want     *describeCli
		wantErr  error
	}{
		{
			name:     "desc preview command",
			kubectl:  kubectlWithNamespace,
			fzfQuery: "",
			want: &describeCli{
				kubectl:   kubectlWithNamespace,
				fzfOption: fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s", "kubectl describe pods {1} -n default", defaultFzfBindOption),
			},
			wantErr: nil,
		},
		{
			name:     "no namespace",
			kubectl:  kubectlWithoutNamespace,
			fzfQuery: "",
			want: &describeCli{
				kubectl:   kubectlWithoutNamespace,
				fzfOption: fmt.Sprintf("--inline-info --layout reverse --preview '%s' --preview-window down:70%% --header-lines 1 --bind %s", "kubectl describe svc {1}", defaultFzfBindOption),
			},
			wantErr: nil,
		},
		{
			name: "KUBECTL_FZF_FZF_OPTION includes invalid env",
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
			got, gotErr := NewDescribeCli(tc.kubectl, tc.fzfQuery)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestDescribeCli_Run(t *testing.T) {
	fzfOption := "--inline-info"
	defaultRunCommand := func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
		assert.Contains(t, commandLine, fmt.Sprintf("| fzf %s", fzfOption))
		return bytes.NewBufferString("pod 2/2 Running 2d").Bytes(), nil
	}
	defaultWantErr := errors.New("want error")
	exitErr := exec.ExitError{}

	testCases := []struct {
		name               string
		runCommandWithFzf  func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error)
		kubectlGetErr      error
		kubectlDescribeErr error
		wantErr            error
		wantIO             string
		wantIOErr          string
	}{
		{
			name:              "output",
			runCommandWithFzf: defaultRunCommand,
			wantErr:           nil,
			wantIO:            "Name: pod",
			wantIOErr:         "",
		},
		{
			name: "command with fzf error",
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, defaultWantErr
			},
			wantErr:   defaultWantErr,
			wantIO:    "",
			wantIOErr: "",
		},
		{
			name: "command with fzf exit error (not 130)",
			runCommandWithFzf: func(ctx context.Context, commandLine string, ioIn io.Reader, ioErr io.Writer) (i []byte, e error) {
				return nil, &exitErr
			},
			wantErr:   &exitErr,
			wantIO:    "",
			wantIOErr: "",
		},
		{
			name:              "kubectl get command error",
			runCommandWithFzf: defaultRunCommand,
			kubectlGetErr:     defaultWantErr,
			wantErr:           defaultWantErr,
			wantIO:            "",
			wantIOErr:         "",
		},
		{
			name:               "kubectl describe command error",
			runCommandWithFzf:  defaultRunCommand,
			kubectlDescribeErr: defaultWantErr,
			wantErr:            defaultWantErr,
			wantIO:             "",
			wantIOErr:          "",
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
				Return([]byte("Name: pod"), tc.kubectlDescribeErr).
				MaxTimes(1)
			runCommandWithFzf = tc.runCommandWithFzf

			sut := describeCli{
				kubectl:   mockKubectl,
				fzfOption: fzfOption,
			}

			var gotIOOut bytes.Buffer
			var gotIOErr bytes.Buffer
			gotErr := sut.Run(context.Background(), strings.NewReader("in"), &gotIOOut, &gotIOErr)
			assert.True(t, errors.Is(gotErr, tc.wantErr), fmt.Sprintf("%+v", gotErr))
			assert.Equal(t, tc.wantIO, gotIOOut.String())
			assert.Equal(t, tc.wantIOErr, gotIOErr.String())
		})
	}
}
