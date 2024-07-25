package anypipe

import (
	"fmt"
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

// TODO @Miguel : RUN function should return result of all steps that executed
//   - steps that did not even execute
//
// should also keep track of how long each step takes to run
func (j *JobImpl) Run(log *slog.Logger,
	du dockerutils.DockerUtils,
	variables map[string]interface{}) error {

	log.Info(fmt.Sprintf("starting job %s", j.Name))

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
