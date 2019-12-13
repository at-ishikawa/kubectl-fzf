package command

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

const (
	kubernetesResourcePods    = "pods"
	kubernetesResourceService = "svc"
)

func TestNewGetCommand(t *testing.T) {
	testCases := []struct {
		name           string
		resource       string
		previewCommand string
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
