package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNewKubectl(t *testing.T) {
	testCases := []struct {
		name      string
		resource  string
		namespace string
		want      *kubectl
		wantErr   error
	}{
		{
			name:      "resource with namespace",
			resource:  kubernetesResourcePods,
			namespace: "default",
			want: &kubectl{
				resource:  kubernetesResourcePods,
				namespace: "default",
			},
		},
		{
			name:     "no namespace",
			resource: kubernetesResourcePods,
			want: &kubectl{
				resource: kubernetesResourcePods,
			},
		},
		{
			name:      "no resource",
			namespace: "default",
			wantErr:   errorInvalidArgumentKubernetesResource,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := NewKubectl(tc.resource, tc.namespace)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestKubectl_getCommand(t *testing.T) {
	testCases := []struct {
		name         string
		kubectl      kubectl
		operation    string
		resourceName string
		options      map[string]string
		want         string
	}{
		{
			name: "resource with namespace",
			kubectl: kubectl{
				resource:  kubernetesResourcePods,
				namespace: "default",
			},
			operation:    "get",
			resourceName: "pod1",
			options: map[string]string{
				"-o": "yaml",
			},
			want: "kubectl get pods pod1 -n default -o yaml",
		},
		{
			name: "no namespace",
			kubectl: kubectl{
				resource: kubernetesResourcePods,
			},
			operation:    "get",
			resourceName: "pod2",
			want:         "kubectl get pods pod2",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.kubectl.getCommand(tc.operation, tc.resourceName, tc.options)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestKubectl_getArguments(t *testing.T) {
	testCases := []struct {
		name         string
		kubectl      kubectl
		operation    string
		resourceName string
		options      map[string]string
		want         []string
	}{
		{
			name: "resource with namespace",
			kubectl: kubectl{
				resource:  kubernetesResourcePods,
				namespace: "default",
			},
			operation:    "get",
			resourceName: "pod1",
			options: map[string]string{
				"-o": "yaml",
			},
			want: []string{
				"get",
				"pods",
				"pod1",
				"-n",
				"default",
				"-o",
				"yaml",
			},
		},
		{
			name: "no namespace",
			kubectl: kubectl{
				resource: kubernetesResourcePods,
			},
			operation:    "get",
			resourceName: "pod2",
			want: []string{
				"get",
				"pods",
				"pod2",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.kubectl.getArguments(tc.operation, tc.resourceName, tc.options)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestKubectl_Run(t *testing.T) {
	backupRunKubectlFunc := runKubectl
	defer func() {
		runKubectl = backupRunKubectlFunc
	}()

	defaultErr := errors.New("error")
	testCases := []struct {
		name         string
		kubectl      kubectl
		operation    string
		resourceName string
		options      map[string]string
		want         []byte
		wantErr      error
	}{
		{
			name: "no error",
			kubectl: kubectl{
				resource:  kubernetesResourcePods,
				namespace: "default",
			},
			operation:    "get",
			resourceName: "pod1",
			options: map[string]string{
				"-o": "yaml",
			},
			want: []byte("pods"),
		},
		{
			name: "with error",
			kubectl: kubectl{
				resource: kubernetesResourcePods,
			},
			operation:    "get",
			resourceName: "pod2",
			want:         nil,
			wantErr:      defaultErr,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runKubectl = func(ctx context.Context, args []string) (bytes []byte, err error) {
				return tc.want, tc.wantErr
			}

			got, gotErr := tc.kubectl.run(context.Background(), tc.operation, tc.resourceName, tc.options)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, errors.Unwrap(gotErr))
		})
	}
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
