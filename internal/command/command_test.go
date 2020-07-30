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
		name          string
		kubectl       kubectl
		operation     string
		resource      string
		resourceNames []string
		options       map[string]string
		want          string
	}{
		{
			name: "resource with namespace",
			kubectl: kubectl{
				namespace: "default",
			},
			operation: "get",
			resource:  kubernetesResourcePods,
			resourceNames: []string{
				"pod1",
				"pod2",
			},
			options: map[string]string{
				"-o": "yaml",
			},
			want: "kubectl get pods pod1 pod2 -n=default -o=yaml",
		},
		{
			name:      "no namespace",
			kubectl:   kubectl{},
			operation: "get",
			resource:  kubernetesResourcePods,
			resourceNames: []string{
				"pod1",
			},
			want: "kubectl get pods pod1",
		},
		{
			name:      "no resource",
			kubectl:   kubectl{},
			operation: "describe",
			resource:  "",
			resourceNames: []string{
				"pod/pod1",
			},
			want: "kubectl describe pod/pod1",
		},
		{
			name:          "no resource names",
			kubectl:       kubectl{},
			operation:     "get",
			resource:      kubernetesResourcePods,
			resourceNames: nil,
			want:          "kubectl get pods",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.kubectl.getCommand(tc.operation, tc.resource, tc.resourceNames, tc.options)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestKubectl_getArguments(t *testing.T) {
	testCases := []struct {
		name          string
		kubectl       kubectl
		operation     string
		resource      string
		resourceNames []string
		options       map[string]string
		want          []string
	}{
		{
			name: "resource with namespace",
			kubectl: kubectl{
				namespace: "default",
			},
			operation: "get",
			resource:  kubernetesResourcePods,
			resourceNames: []string{
				"pod1",
				"pod2",
			},
			options: map[string]string{
				"-o": "yaml",
			},
			want: []string{
				"get",
				"pods",
				"pod1",
				"pod2",
				"-n=default",
				"-o=yaml",
			},
		},
		{
			name: "no resource with namespace",
			kubectl: kubectl{
				namespace: "default",
			},
			operation: "get",
			resource:  "",
			resourceNames: []string{
				"pod/pod1",
				"svc/svc2",
			},
			options: map[string]string{
				"-o": "yaml",
			},
			want: []string{
				"get",
				"pod/pod1",
				"svc/svc2",
				"-n=default",
				"-o=yaml",
			},
		},
		{
			name:      "no namespace",
			kubectl:   kubectl{},
			operation: "get",
			resource:  kubernetesResourcePods,
			resourceNames: []string{
				"pod1",
			},
			want: []string{
				"get",
				"pods",
				"pod1",
			},
		},
		{
			name:      "no resource names",
			kubectl:   kubectl{},
			operation: "get",
			resource:  kubernetesResourcePods,
			want: []string{
				"get",
				"pods",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.kubectl.getArguments(tc.operation, tc.resource, tc.resourceNames, tc.options)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestKubectl_Run(t *testing.T) {
	backupRunKubectlFunc := runKubectl
	defer func() {
		runKubectl = backupRunKubectlFunc
	}()

	// defaultErr := errors.New("error")
	testCases := []struct {
		name          string
		kubectl       kubectl
		operation     string
		resourceNames []string
		options       map[string]string
		kubectlOut    []byte
		kubectlErr    error
		want          []byte
		wantErr       error
	}{
		{
			name: "no error",
			kubectl: kubectl{
				resource:  kubernetesResourcePods,
				namespace: "default",
			},
			operation:     "get",
			resourceNames: []string{"pod1"},
			options: map[string]string{
				"-o": "yaml",
			},
			kubectlOut: []byte("pods"),
			want:       []byte("pods"),
		},
		{
			name: "error with stdout",
			kubectl: kubectl{
				resource: kubernetesResourcePods,
			},
			operation:     "get",
			resourceNames: []string{"pod2"},
			kubectlOut:    []byte("server doesn't have a resource type"),
			kubectlErr:    errors.New("exit status: 1"),
			want:          nil,
			wantErr:       errors.New("server doesn't have a resource type"),
		},
		{
			name: "error without stdout",
			kubectl: kubectl{
				resource: kubernetesResourcePods,
			},
			operation:     "get",
			resourceNames: []string{"pod2"},
			kubectlOut:    nil,
			kubectlErr:    errors.New("k executable file not found"),
			want:          nil,
			wantErr:       errors.New("k executable file not found"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runKubectl = func(ctx context.Context, args []string) (bytes []byte, err error) {
				return tc.kubectlOut, tc.kubectlErr
			}

			got, gotErr := tc.kubectl.run(context.Background(), tc.operation, tc.resourceNames, tc.options)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}

func TestGetFzfOption(t *testing.T) {
	testCases := []struct {
		name                 string
		previewCommand       string
		hasMultipleResources bool
		envVars              map[string]string
		want                 string
		wantErr              error
	}{
		{
			name:                 "no env vars for multiple resources",
			previewCommand:       "kubectl describe {{1}}",
			hasMultipleResources: true,
			want:                 fmt.Sprintf("--inline-info --multi --layout reverse --preview '%s' --preview-window down:70%% --bind ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down", "kubectl describe {{1}}"),
		},
		{
			name:           "no env vars for single resource",
			previewCommand: "kubectl describe pods {{1}}",
			want:           fmt.Sprintf("--inline-info --multi --layout reverse --preview '%s' --preview-window down:70%% --bind ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down --header-lines 1", "kubectl describe pods {{1}}"),
		},
		{
			name:           "all correct env vars",
			previewCommand: "kubectl describe pods {{1}}",
			envVars: map[string]string{
				envNameFzfOption: "--preview '$KUBECTL_FZF_FZF_PREVIEW_OPTION' --bind ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down",
			},
			want: fmt.Sprintf("--preview '%s' --bind %s", "kubectl describe pods {{1}}", "ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down"),
		},
		{
			name:           "no preview command in environment variable",
			previewCommand: "unused preview command",
			envVars: map[string]string{
				envNameFzfOption: "--inline-info",
			},
			want: "--inline-info",
		},
		{
			name:           "invalid env vars in KUBECTL_FZF_FZF_OPTION",
			previewCommand: "unused preview command",
			envVars: map[string]string{
				envNameFzfOption: "--inline-info $UNKNOWN_ENV_NAME",
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
			got, gotErr := getFzfOption(tc.previewCommand, tc.hasMultipleResources)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantErr, gotErr)
		})
	}
}
