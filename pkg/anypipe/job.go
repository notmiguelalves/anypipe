package anypipe

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
)

type Job interface {
	WithStep(stepName string, f StepFunc) Job
	Run(log *slog.Logger, du dockerutils.DockerUtils, variables map[string]interface{}) error
	DisplaySummary()
}

type StepMetrics struct {
	StepName string
	Duration time.Duration
	Result   error
}

type JobImpl struct {
	Name     string
	ImageRef string
	Steps    []Step
	Metrics  []StepMetrics
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

	log.Info(fmt.Sprintf("starting job %s", j.Name))

	c, err := du.CreateContainer(j.ImageRef)
	if err != nil {
		return err
	}

	gotError := false
	for _, step := range j.Steps {
		if gotError {
			// mark step as skipped
			j.Metrics = append(j.Metrics, StepMetrics{
				StepName: step.GetName(),
				Duration: time.Duration(0),
				Result:   errors.New("SKIPPED"),
			})
			continue
		}
		startTime := time.Now()
		err := step.Run(log, du, c, variables)
		endTime := time.Now()
		stepDuration := endTime.Sub(startTime)
		if err != nil {
			gotError = true
		}

		j.Metrics = append(j.Metrics, StepMetrics{
			StepName: step.GetName(),
			Duration: stepDuration,
			Result:   err,
		})
	}

	return nil
}

func (j *JobImpl) DisplaySummary() {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle(j.Name)
	t.AppendHeader(table.Row{"Result", "Step", "Duration"})

	for _, m := range j.Metrics {
		res := "PASS"
		if m.Result != nil {
			if strings.Contains(m.Result.Error(), "SKIPPED") {
				res = "SKIP"
			} else {
				res = "FAIL"
			}
		}
		t.AppendRow(table.Row{res, m.StepName, fmt.Sprintf("%s", m.Duration)})
	}
	t.Render()

	if len(os.Getenv("GITHUB_ACTIONS")) > 0 {
		githubSummary, err := os.OpenFile(os.Getenv("GITHUB_STEP_SUMMARY"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return
		}

		t.SetOutputMirror(githubSummary)
		t.RenderMarkdown()
	}
}
