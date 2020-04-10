package suggestion_goptuna_v1alpha3

import (
	"errors"
	"strconv"
	"time"

	"github.com/c-bata/goptuna"
	"github.com/c-bata/goptuna/cmaes"
	"github.com/c-bata/goptuna/tpe"
	api_v1_alpha3 "github.com/kubeflow/katib/pkg/apis/manager/v1alpha3"
)

func toGoptunaDirection(t api_v1_alpha3.ObjectiveType) (goptuna.StudyDirection, error) {
	if t == api_v1_alpha3.ObjectiveType_MINIMIZE {
		return goptuna.StudyDirectionMinimize, nil
	} else if t == api_v1_alpha3.ObjectiveType_MAXIMIZE {
		return goptuna.StudyDirectionMaximize, nil
	}
	return "", errors.New("unexpected objective type")
}

func toGoptunaSampler(algorithm *api_v1_alpha3.AlgorithmSpec) (goptuna.Sampler, goptuna.RelativeSampler, error) {
	name := algorithm.GetAlgorithmName()
	if name == AlgorithmCMAES {
		opts := make([]cmaes.SamplerOption, 0, len(algorithm.GetAlgorithmSetting()))
		for _, s := range algorithm.GetAlgorithmSetting() {
			if s.Name == "random_state" {
				seed, err := strconv.Atoi(s.Value)
				if err != nil {
					return nil, nil, err
				}
				opts = append(opts, cmaes.SamplerOptionSeed(int64(seed)))
			}
		}
		return nil, cmaes.NewSampler(opts...), nil
	} else if name == AlgorithmTPE {
		opts := make([]tpe.SamplerOption, 0, len(algorithm.GetAlgorithmSetting()))
		for _, s := range algorithm.GetAlgorithmSetting() {
			if s.Name == "random_state" {
				seed, err := strconv.Atoi(s.Value)
				if err != nil {
					return nil, nil, err
				}
				opts = append(opts, tpe.SamplerOptionSeed(int64(seed)))
			}
		}
		return tpe.NewSampler(opts...), nil, nil
	} else {
		opts := make([]goptuna.RandomSearchSamplerOption, 0, len(algorithm.GetAlgorithmSetting()))
		for _, s := range algorithm.GetAlgorithmSetting() {
			if s.Name == "random_state" {
				seed, err := strconv.Atoi(s.Value)
				if err != nil {
					return nil, nil, err
				}
				opts = append(opts, goptuna.RandomSearchSamplerOptionSeed(int64(seed)))
			}
		}
		return goptuna.NewRandomSearchSampler(opts...), nil, nil
	}
}

func toGoptunaSearchSpace(parameters []*api_v1_alpha3.ParameterSpec) (map[string]interface{}, error) {
	searchSpace := make(map[string]interface{}, len(parameters))
	for _, p := range parameters {
		if p.ParameterType == api_v1_alpha3.ParameterType_UNKNOWN_TYPE {
			return nil, errors.New("invalid parameter type")
		}

		if p.ParameterType == api_v1_alpha3.ParameterType_DOUBLE {
			high, err := strconv.ParseFloat(p.GetFeasibleSpace().GetMax(), 64)
			if err != nil {
				return nil, err
			}
			low, err := strconv.ParseFloat(p.GetFeasibleSpace().GetMin(), 64)
			if err != nil {
				return nil, err
			}

			stepstr := p.GetFeasibleSpace().GetStep()
			if stepstr == "" {
				searchSpace[p.Name] = goptuna.UniformDistribution{
					High: high,
					Low:  low,
				}
			} else {
				step, err := strconv.ParseFloat(stepstr, 64)
				if err != nil {
					return nil, err
				}
				searchSpace[p.Name] = goptuna.DiscreteUniformDistribution{
					High: high,
					Low:  low,
					Q:    step,
				}
			}
		} else if p.ParameterType == api_v1_alpha3.ParameterType_INT {
			high, err := strconv.Atoi(p.GetFeasibleSpace().GetMax())
			if err != nil {
				return nil, err
			}
			low, err := strconv.Atoi(p.GetFeasibleSpace().GetMin())
			if err != nil {
				return nil, err
			}
			stepstr := p.GetFeasibleSpace().GetStep()
			if stepstr == "" {
				searchSpace[p.Name] = goptuna.IntUniformDistribution{
					High: high,
					Low:  low,
				}
			} else {
				step, err := strconv.Atoi(stepstr)
				if err != nil {
					return nil, err
				}
				searchSpace[p.Name] = goptuna.StepIntUniformDistribution{
					High: high,
					Low:  low,
					Step: step,
				}
			}
		} else if p.ParameterType == api_v1_alpha3.ParameterType_CATEGORICAL {
			choices := p.GetFeasibleSpace().GetList()
			searchSpace[p.Name] = goptuna.CategoricalDistribution{
				Choices: choices,
			}
		} else if p.ParameterType == api_v1_alpha3.ParameterType_DISCRETE {
			// Use categorical distribution instead of goptuna.DiscreteUniformDistribution
			// because goptuna.UniformDistributions needs to declare the parameter space
			// with minimum value, maximum value and interval.
			choices := p.GetFeasibleSpace().GetList()
			searchSpace[p.Name] = goptuna.CategoricalDistribution{
				Choices: choices,
			}
		} else {
			return nil, errors.New("unsupported parameter type")
		}
	}
	return searchSpace, nil
}

func toGoptunaState(condition api_v1_alpha3.TrialStatus_TrialConditionType) (goptuna.TrialState, error) {
	if condition == api_v1_alpha3.TrialStatus_CREATED {
		return goptuna.TrialStateWaiting, nil
	} else if condition == api_v1_alpha3.TrialStatus_RUNNING {
		return goptuna.TrialStateRunning, nil
	} else if condition == api_v1_alpha3.TrialStatus_SUCCEEDED {
		return goptuna.TrialStateComplete, nil
	} else if condition == api_v1_alpha3.TrialStatus_KILLED {
		return goptuna.TrialStateFail, nil
	} else if condition == api_v1_alpha3.TrialStatus_FAILED {
		return goptuna.TrialStateFail, nil
	}
	return goptuna.TrialStateFail, errors.New("unexpected trial condition")
}

func toGoptunaTrials(
	ktrials []*api_v1_alpha3.Trial,
	objectMetricName string,
	study *goptuna.Study,
	searchSpace map[string]interface{},
) ([]goptuna.FrozenTrial, error) {
	gtrials := make([]goptuna.FrozenTrial, 0, len(ktrials))
	for i, kt := range ktrials {
		datetimeStart, err := time.Parse(time.RFC3339Nano, kt.GetStatus().GetStartTime())
		if err != nil {
			return nil, err
		}
		datetimeComplete, err := time.Parse(time.RFC3339Nano, kt.GetStatus().GetCompletionTime())
		if err != nil {
			return nil, err
		}
		state, err := toGoptunaState(kt.GetStatus().GetCondition())
		if err != nil {
			return nil, err
		}

		metrics := kt.GetStatus().GetObservation().GetMetrics()
		var finalValue float64
		if state == goptuna.TrialStateComplete {
			for i := len(metrics) - 1; i >= 0; i-- {
				if metrics[i].GetName() != objectMetricName {
					continue
				}
				v, err := strconv.ParseFloat(metrics[i].GetValue(), 64)
				if err != nil {
					return nil, err
				}
				finalValue = v
				break
			}
		}

		assignments := kt.GetSpec().GetParameterAssignments().GetAssignments()
		internalParams, externalParams, err := toGoptunaParams(assignments, searchSpace)
		if err != nil {
			return nil, err
		}

		gt := goptuna.FrozenTrial{
			ID:                 i, // dummy id
			StudyID:            study.ID,
			Number:             i, // dummy number
			State:              state,
			Value:              finalValue,
			IntermediateValues: nil,
			DatetimeStart:      datetimeStart,
			DatetimeComplete:   datetimeComplete,
			InternalParams:     internalParams,
			Params:             externalParams,
			Distributions:      searchSpace,
			UserAttrs:          nil,
			SystemAttrs:        nil,
		}
		gtrials = append(gtrials, gt)
	}
	return gtrials, nil
}

func toGoptunaParams(
	assignments []*api_v1_alpha3.ParameterAssignment,
	searchSpace map[string]interface{},
) (
	internalParams map[string]float64,
	externalParams map[string]interface{},
	err error,
) {
	internalParams = make(map[string]float64, len(assignments))
	externalParams = make(map[string]interface{}, len(assignments))

	for i := range assignments {
		name := assignments[i].GetName()
		valueStr := assignments[i].GetValue()

		switch d := searchSpace[name].(type) {
		case goptuna.UniformDistribution:
			p, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return nil, nil, err
			}
			internalParams[name] = p
			externalParams[name] = d.ToExternalRepr(p)
		case goptuna.DiscreteUniformDistribution:
			p, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return nil, nil, err
			}
			internalParams[name] = p
			externalParams[name] = d.ToExternalRepr(p)
		case goptuna.IntUniformDistribution:
			p, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return nil, nil, err
			}
			internalParams[name] = float64(p)
			externalParams[name] = p
		case goptuna.CategoricalDistribution:
			internalRepr := -1.0
			for i := range d.Choices {
				if d.Choices[i] == valueStr {
					internalRepr = float64(i)
					break
				}
			}
			if internalRepr == -1.0 {
				return nil, nil, errors.New("invalid categorical value")
			}
			internalParams[name] = internalRepr
			externalParams[name] = valueStr
		}
	}
	return internalParams, externalParams, nil
}