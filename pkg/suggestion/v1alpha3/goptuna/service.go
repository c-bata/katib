package suggestion_goptuna_v1alpha3

import (
	"context"
	"errors"

	"github.com/c-bata/goptuna"
	"github.com/kubeflow/katib/pkg/apis/manager/v1alpha3"
	"k8s.io/klog"
)

const (
	AlgorithmCMAES  = "cmaes"
	AlgorithmTPE    = "tpe"
	AlgorithmRandom = "random"
)

func NewSuggestionService() *SuggestionService {
	return &SuggestionService{}
}

type SuggestionService struct{}

func (s *SuggestionService) GetSuggestions(
	ctx context.Context,
	req *api_v1_alpha3.GetSuggestionsRequest,
) (*api_v1_alpha3.GetSuggestionsReply, error) {
	if req == nil {
		klog.Errorf("Empty request.")
		return nil, errors.New("invalid request")
	}

	direction, err := toGoptunaDirection(req.GetExperiment().GetSpec().GetObjective().GetType())
	if err != nil {
		klog.Errorf("Failed to convert to Goptuna direction: %s", err)
		return nil, err
	}
	independentSampler, relativeSampler, err := toGoptunaSampler(req.GetExperiment().GetSpec().GetAlgorithm())
	if err != nil {
		klog.Errorf("Failed to create Goptuna sampler: %s", err)
		return nil, err
	}
	searchSpace, err := toGoptunaSearchSpace(req.GetExperiment().GetSpec().GetParameterSpecs().GetParameters())
	if err != nil {
		klog.Errorf("Failed to convert to Goptuna search space: %s", err)
		return nil, err
	}
	klog.Infof("Goptuna search space: %#v", searchSpace)

	studyOpts := make([]goptuna.StudyOption, 0, 3)
	studyOpts = append(studyOpts, goptuna.StudyOptionSetDirection(direction))
	if independentSampler != nil {
		studyOpts = append(studyOpts, goptuna.StudyOptionSampler(independentSampler))
	}
	if relativeSampler != nil {
		studyOpts = append(studyOpts, goptuna.StudyOptionRelativeSampler(relativeSampler))
	}

	study, err := goptuna.CreateStudy("katib", studyOpts...)
	if err != nil {
		klog.Errorf("Failed to create Goptuna study: %s", err)
		return nil, err
	}
	trials, err := toGoptunaTrials(req.GetTrials(), study, searchSpace)
	if err != nil {
		klog.Errorf("Failed to convert to Goptuna trials: %s", err)
		return nil, err
	}
	for _, t := range trials {
		_, err = study.Storage.CloneTrial(study.ID, t)
		if err != nil {
			klog.Errorf("Failed to register trials: %s", err)
			return nil, err
		}
	}

	requestNumber := int(req.GetRequestNumber())
	parameterAssignments := make([]*api_v1_alpha3.GetSuggestionsReply_ParameterAssignments, requestNumber)
	for i := 0; i < requestNumber; i++ {
		assignments, err := sampleNextParam(study, searchSpace)
		if err != nil {
			klog.Errorf("Failed to sample next param: %s", err)
			return nil, err
		}
		parameterAssignments[i] = &api_v1_alpha3.GetSuggestionsReply_ParameterAssignments{
			Assignments: assignments,
		}
	}

	klog.Infof("Success to sample %d parameters", requestNumber)
	return &api_v1_alpha3.GetSuggestionsReply{
		ParameterAssignments: parameterAssignments,
		Algorithm: &api_v1_alpha3.AlgorithmSpec{
			AlgorithmName:     "",
			AlgorithmSetting:  nil,
			EarlyStoppingSpec: &api_v1_alpha3.EarlyStoppingSpec{},
		},
	}, nil
}

func (s *SuggestionService) ValidateAlgorithmSettings(ctx context.Context, req *api_v1_alpha3.ValidateAlgorithmSettingsRequest) (*api_v1_alpha3.ValidateAlgorithmSettingsReply, error) {
	return &api_v1_alpha3.ValidateAlgorithmSettingsReply{}, nil
}

// This is a compile-time assertion to ensure that SuggestionService
// implements an api_v1_alpha3.SuggestionServer interface.
var _ api_v1_alpha3.SuggestionServer = &SuggestionService{}
