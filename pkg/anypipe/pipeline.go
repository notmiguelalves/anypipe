package anypipe

import (
	"context"
	"log/slog"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
)

type Anypipe interface {
	WithJob(job Job) Anypipe
	Run(inputs map[string]interface{}) error
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

func (p *AnypipeImpl) WithJob(job Job) Anypipe {
	p.Jobs = append(p.Jobs, job)

	return p
}

func (p *AnypipeImpl) Run(inputs map[string]interface{}) error {
	du, err := dockerutils.New(p.ctx, p.log)
	if err != nil {
		return err
	}
	defer du.Close()

	nextInputs := inputs
	for _, job := range p.Jobs {
		outputs, err := job.Run(p.log, du, nextInputs)
		if err != nil {
			return err
		}
		nextInputs = outputs
	}

	return nil
}
