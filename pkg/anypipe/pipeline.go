package anypipe

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
)

type Anypipe interface {
	WithSequentialJobs(jobs ...Job) Anypipe
	Run(variables map[string]interface{}) error
}

type AnypipeImpl struct {
	Name string
	Jobs []Job
	ctx  context.Context
	log  *slog.Logger
}

func NewPipelineImpl(ctx context.Context, log *slog.Logger, name string) Anypipe {
	return &AnypipeImpl{
		Name: name,
		Jobs: []Job{},
		ctx:  ctx,
		log:  log,
	}
}

func (p *AnypipeImpl) WithSequentialJobs(jobs ...Job) Anypipe {
	p.Jobs = append(p.Jobs, jobs...)

	return p
}

// TODO @Miguel : should print to stdout (and github summary) overview
// of executed steps
func (p *AnypipeImpl) Run(variables map[string]interface{}) error {
	p.log.Info(fmt.Sprintf("starting pipeline %s", p.Name))
	du, err := dockerutils.New(p.ctx, p.log)
	if err != nil {
		return err
	}
	defer du.Close()

	for _, job := range p.Jobs {
		err := job.Run(p.log, du, variables)
		if err != nil {
			return err
		}
	}

	return nil
}
