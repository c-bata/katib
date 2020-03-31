package suggestion_goptuna_v1alpha3

import (
	"testing"

	"github.com/c-bata/goptuna"
	api_v1_alpha3 "github.com/kubeflow/katib/pkg/apis/manager/v1alpha3"
)

func Test_toGoptunaDirection(t *testing.T) {
	for _, tt := range []struct {
		name          string
		objectiveType api_v1_alpha3.ObjectiveType
		expected      goptuna.StudyDirection
		wantErr       bool
	}{
		{
			name:          "minimize",
			objectiveType: api_v1_alpha3.ObjectiveType_MINIMIZE,
			expected:      goptuna.StudyDirectionMinimize,
			wantErr:       false,
		},
		{
			name:          "maximize",
			objectiveType: api_v1_alpha3.ObjectiveType_MAXIMIZE,
			expected:      goptuna.StudyDirectionMaximize,
			wantErr:       false,
		},
		{
			name:          "unexpected objective type",
			objectiveType: api_v1_alpha3.ObjectiveType_UNKNOWN,
			expected:      "",
			wantErr:       true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toGoptunaDirection(tt.objectiveType)
			if (err != nil) != tt.wantErr {
				t.Errorf("toGoptunaDirection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("toGoptunaDirection() got = %v, want %v", got, tt.expected)
			}
		})
	}
}
