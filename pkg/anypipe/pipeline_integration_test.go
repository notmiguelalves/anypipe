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

	f1 := func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error {
		variables["out_file"] = "f1.txt"

		_, _, _, err := du.Exec(c, "echo 'test data' > f1.txt")
		return err
	}

	f2 := func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error {
		file, ok := variables["out_file"].(string)
		if !ok {
			return errors.New("unable to cast 'out_file' to string")
		}

		stdout, _, _, err := du.Exec(c, fmt.Sprintf("cat %s", file))
		if err != nil {
			return err
		}

		if !strings.Contains(stdout, "test data") {
			return errors.New("expected file to contain 'test data'")
		}

		return nil
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pipeline := NewPipelineImpl(ctx, logger, "test_pipeline")

	pipeline.WithSequentialJobs(
		NewJobImpl("test_job_1", "alpine:latest").
			WithStep("step1", f1).
			WithStep("step2", f2),
	)

	err := pipeline.Run(map[string]interface{}{})
	assert.NoError(t, err)
}
