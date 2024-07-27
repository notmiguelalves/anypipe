package anypipe

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	du := dockerutils.NewMockDockerUtils(ctrl)
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	f1 := func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error {
		variables["TESTVAR"] = "TESTVALUE"

		return nil
	}

	f2 := func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error {
		val, ok := variables["TESTVAR"]
		assert.True(t, ok)

		stdout, _, ec, err := du.Exec(c, fmt.Sprintf("echo '%s'", val.(string)))
		assert.NoError(t, err)
		assert.Equal(t, 0, ec)
		assert.Contains(t, stdout.String(), "TESTVALUE")

		return nil
	}

	du.EXPECT().CreateContainer("testimage:latest").Times(1).Return(&dockerutils.Container{}, nil)
	du.EXPECT().Exec(gomock.Any(), "echo 'TESTVALUE'").Times(1).Return(bytes.NewBufferString("TESTVALUE"), bytes.NewBufferString(""), 0, nil)

	job := NewJobImpl("test job", "testimage:latest").
		WithStep("step1", f1).
		WithStep("step2", f2)

	err := job.Run(testLogger, du, map[string]interface{}{})
	assert.NoError(t, err)
}

func TestJobWithFailedSteps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	du := dockerutils.NewMockDockerUtils(ctrl)
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	f1 := func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error {
		return errors.New("some error")
	}

	f2 := func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error {
		return nil
	}

	du.EXPECT().CreateContainer("testimage:latest").Times(1).Return(&dockerutils.Container{}, nil)

	job := NewJobImpl("bad job", "testimage:latest").
		WithStep("step1", f1).
		WithStep("step2", f2)

	err := job.Run(testLogger, du, map[string]interface{}{})
	assert.Error(t, err)
	job.DisplaySummary()
}
