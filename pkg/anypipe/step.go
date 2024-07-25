package anypipe

import (
	"fmt"
	"log/slog"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
)

type StepFunc func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error

type Step interface {
	Run(log *slog.Logger, du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error
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

func (s *StepImpl) Run(log *slog.Logger,
	du dockerutils.DockerUtils,
	c *dockerutils.Container,
	variables map[string]interface{}) error {

	log.Info(fmt.Sprintf("running step %s", s.Name))
	return s.Impl(du, c, variables)
}
