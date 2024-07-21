package utils

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTarUntar(t *testing.T) {
	bbuf, err := Tar("testdata/tmp.txt")
	assert.NoError(t, err)

	assert.NoError(t, os.Mkdir("testdata/testuntar", os.ModePerm))
	defer os.RemoveAll("testdata/testuntar")

	assert.NoError(t, Untar(io.NopCloser(bbuf), "testdata/testuntar/tmp.txt"))

	res, err := os.ReadFile("testdata/testuntar/tmp.txt")
	assert.NoError(t, err)
	assert.Equal(t, "this is a test string", string(res))
}
