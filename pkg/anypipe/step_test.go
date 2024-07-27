package anypipe

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestStep(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	du := dockerutils.NewMockDockerUtils(ctrl)
	c := dockerutils.Container{}
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	f1 := func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error {
		val, ok := variables["TESTVAR"]
		assert.True(t, ok)

		stdout, _, ec, err := du.Exec(c, fmt.Sprintf("echo '%s'", val.(string)))
		assert.NoError(t, err)
		assert.Equal(t, 0, ec)
		assert.Contains(t, stdout.String(), "TESTVALUE")

		return nil
	}

	du.EXPECT().Exec(&c, "echo 'TESTVALUE'").Times(1).Return(bytes.NewBufferString("TESTVALUE"), bytes.NewBufferString(""), 0, nil)

	step := NewStepImpl("test step", f1)

	err := step.Run(testLogger, du, &c, map[string]interface{}{"TESTVAR": "TESTVALUE"})
	assert.NoError(t, err)
}
