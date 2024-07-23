package anypipe

import (
	"log/slog"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
)

type StepFunc func(du dockerutils.DockerUtils, c *dockerutils.Container, inputs map[string]interface{}) (outputs map[string]interface{}, err error)

type Step interface {
	Run(log *slog.Logger, du dockerutils.DockerUtils, c *dockerutils.Container, inputs map[string]interface{}) (outputs map[string]interface{}, err error)
}

type StepImpl struct {
	Name string
	Impl StepFunc
}

func NewStepImpl(name string, impl StepFunc) Step {
	return &StepImpl{
		Name: name,
		Impl: impl,
	}
}

// TODO @Miguel : instead of inputs and outputs, lets just have a map for 'shared' stuff, that is passed
// as argument via pointer

func (s *StepImpl) Run(log *slog.Logger,
	du dockerutils.DockerUtils,
	c *dockerutils.Container,
	inputs map[string]interface{}) (outputs map[string]interface{}, err error) {

	return s.Impl(du, c, inputs)
}
