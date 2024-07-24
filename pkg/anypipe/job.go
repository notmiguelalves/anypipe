package anypipe

import (
	"log/slog"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
)

type Job interface {
	WithStep(stepName string, f StepFunc) Job
	Run(log *slog.Logger, du dockerutils.DockerUtils, variables map[string]interface{}) error
}

type JobImpl struct {
	Name     string
	ImageRef string
	Steps    []Step
}

func NewJobImpl(name, imageRef string) Job {
	return &JobImpl{
		Name:     name,
		ImageRef: imageRef,
		Steps:    []Step{},
	}
}

func (j *JobImpl) WithStep(stepName string, f StepFunc) Job {
	newStep := NewStepImpl(stepName, f)
	j.Steps = append(j.Steps, newStep)

	return j
}

func (j *JobImpl) Run(log *slog.Logger,
	du dockerutils.DockerUtils,
	variables map[string]interface{}) error {

	c, err := du.CreateContainer(j.ImageRef)
	if err != nil {
		return err
	}

	for _, step := range j.Steps {
		err := step.Run(log, du, c, variables)
		if err != nil {
			return err
		}
	}

	return nil
}
