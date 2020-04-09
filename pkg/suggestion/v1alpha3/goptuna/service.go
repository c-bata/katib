package suggestion_goptuna_v1alpha3

import (
	"context"
	"errors"
	"strconv"

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

	nextTrialID, err := study.Storage.CreateNewTrial(study.ID)
	if err != nil {
		klog.Errorf("Failed to create a new trial: %s", err)
		return nil, err
	}
	nextTrial, err := study.Storage.GetTrial(nextTrialID)
	if err != nil {
		klog.Errorf("Failed to get a next trial: %s", err)
		return nil, err
	}

	var relativeSampleParams map[string]float64
	if relativeSampler != nil {
		relativeSampleParams, err = relativeSampler.SampleRelative(study, nextTrial, searchSpace)
		if err != nil {
			klog.Errorf("Failed to call SampleRelative: %s", err)
			return nil, err
		}
	}

	assignments := make([]*api_v1_alpha3.ParameterAssignment, 0, len(searchSpace))
	trial := goptuna.Trial{
		Study: study,
		ID:    nextTrialID,
	}
	for name := range searchSpace {
		switch distribution := searchSpace[name].(type) {
		case goptuna.UniformDistribution:
			var p float64
			if internalParam, ok := relativeSampleParams[name]; ok {
				p = internalParam
			} else {
				p, err = trial.SuggestUniform(name, distribution.Low, distribution.High)
				if err != nil {
					klog.Errorf("Failed to get suggested param: %s", err)
					return nil, err
				}
			}
			assignments = append(assignments, &api_v1_alpha3.ParameterAssignment{
				Name:  name,
				Value: strconv.FormatFloat(p, 'f', -1, 64),
			})
		case goptuna.IntUniformDistribution:
			var p int
			if internalParam, ok := relativeSampleParams[name]; ok {
				p = int(internalParam)
			} else {
				p, err = trial.SuggestInt(name, distribution.Low, distribution.High)
				if err != nil {
					klog.Errorf("Failed to get suggested param: %s", err)
					return nil, err
				}
			}
			assignments = append(assignments, &api_v1_alpha3.ParameterAssignment{
				Name:  name,
				Value: strconv.Itoa(p),
			})
		case goptuna.DiscreteUniformDistribution:
			var p float64
			if internalParam, ok := relativeSampleParams[name]; ok {
				p = internalParam
			} else {
				p, err = trial.SuggestDiscreteUniform(name, distribution.Low, distribution.High, distribution.Q)
				if err != nil {
					klog.Errorf("Failed to get suggested param: %s", err)
					return nil, err
				}
			}
			assignments = append(assignments, &api_v1_alpha3.ParameterAssignment{
				Name:  name,
				Value: strconv.FormatFloat(p, 'f', -1, 64),
			})
		case goptuna.CategoricalDistribution:
			var p string
			if internalParam, ok := relativeSampleParams[name]; ok {
				p = distribution.Choices[int(internalParam)]
			} else {
				p, err = trial.SuggestCategorical(name, distribution.Choices)
				if err != nil {
					klog.Errorf("Failed to get suggested param: %s", err)
					return nil, err
				}
			}
			assignments = append(assignments, &api_v1_alpha3.ParameterAssignment{
				Name:  name,
				Value: p,
			})
		}
	}
	klog.Infof("Goptuna search space: %#v", searchSpace)
	klog.Infof("Katib assignments: %#v", assignments)

	return &api_v1_alpha3.GetSuggestionsReply{
		ParameterAssignments: []*api_v1_alpha3.GetSuggestionsReply_ParameterAssignments{
			{
				Assignments: assignments,
			},
		},
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
