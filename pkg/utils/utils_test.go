package utils

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTar(t *testing.T) {
	bbuf, err := Tar("testdata/tmp.txt")
	assert.NoError(t, err)

	bbytes, err := io.ReadAll(bbuf)
	assert.NoError(t, err)

	tarbytes, err := os.ReadFile("testdata/tmp.tar")
	assert.NoError(t, err)
	assert.Equal(t, tarbytes, bbytes)
}

func TestUntar(t *testing.T) {
	bbytes, err := os.ReadFile("testdata/tmp.tar")
	assert.NoError(t, err)

	bbuf := bytes.NewBuffer(bbytes)

	assert.NoError(t, os.Mkdir("testdata/testuntar", os.ModePerm))
	defer os.RemoveAll("testdata/testuntar")

	assert.NoError(t, Untar(io.NopCloser(bbuf), "testdata/testuntar/tmp.txt"))

	res, err := os.ReadFile("testdata/testuntar/tmp.txt")
	assert.NoError(t, err)
	assert.Equal(t, "this is a test string", string(res))
}
