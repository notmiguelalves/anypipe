package anypipe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
	"github.com/stretchr/testify/assert"
)

func TestAnypipe(t *testing.T) {

	f1 := func(du dockerutils.DockerUtils, c *dockerutils.Container, inputs map[string]interface{}) (outputs map[string]interface{}, err error) {
		outputs = map[string]interface{}{}

		_, _, _, err = du.Exec(c, "echo 'Hello World!'")
		outputs["test_output"] = "this is an output"

		return
	}

	f2 := func(du dockerutils.DockerUtils, c *dockerutils.Container, inputs map[string]interface{}) (outputs map[string]interface{}, err error) {
		outputs = map[string]interface{}{}

		stdout, _, _, err := du.Exec(c, fmt.Sprintf("echo '%s'", inputs["test_output"].(string)))
		if err != nil {
			return
		}

		if !strings.Contains(stdout.String(), "this is an output") {
			err = errors.New("expected inputs[test_output] to contain 'this is an output'")
		}

		return
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pipeline := NewPipelineImpl(ctx, logger, "test_pipeline")

	pipeline.WithJob(
		NewJobImpl("test_job_1", "alpine:latest").
			WithStep("step1", f1).
			WithStep("step2", f2),
	)

	err := pipeline.Run(map[string]interface{}{})
	assert.NoError(t, err)
}
