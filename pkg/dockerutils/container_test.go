package dockerutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeEnvKey(t *testing.T) {
	type testcase struct {
		input          string
		expectedOutput string
	}

	testcases := []testcase{
		{
			input:          "some key",
			expectedOutput: "somekey",
		},
		{
			input:          "AnoThErKeY",
			expectedOutput: "AnoThErKeY",
		},
		{
			input:          "    A    K   E   Y    ",
			expectedOutput: "AKEY",
		},
		{
			input:          "MYKE Y     ",
			expectedOutput: "MYKEY",
		},
	}

	for _, tc := range testcases {
		res := sanitizeEnvKey(tc.input)
		assert.Equal(t, tc.expectedOutput, res)
	}
}

func TestEnv(t *testing.T) {
	type testcase struct {
		input          Container
		expectedOutput []string
	}

	testcases := []testcase{
		{
			input: Container{
				env: map[string]string{},
			},
			expectedOutput: []string{},
		},
		{
			input: Container{
				env: map[string]string{
					"SOMEKEY": "SOMEVALUE",
				},
			},
			expectedOutput: []string{"SOMEKEY='SOMEVALUE'"},
		},
		{
			input: Container{
				env: map[string]string{
					"SOMEKEY":    "SOMEVALUE",
					"ANOTHERKEY": "ANOTHERVALUE",
				},
			},
			expectedOutput: []string{"SOMEKEY='SOMEVALUE'", "ANOTHERKEY='ANOTHERVALUE'"},
		},
		{
			input: Container{
				env: map[string]string{
					"SOMEKEY": "SOME VALUE WITH SPACES",
				},
			},
			expectedOutput: []string{"SOMEKEY='SOME VALUE WITH SPACES'"},
		},
	}

	for _, tc := range testcases {
		res := tc.input.Env()
		assert.ElementsMatch(t, tc.expectedOutput, res)
	}
}

func TestAddEnv(t *testing.T) {
	c := Container{env: map[string]string{}}

	assert.Empty(t, c.env)

	c.AddEnv("key", "value")
	assert.Equal(t, "value", c.env["key"])

	c.AddEnv("another key", "another value")
	assert.Equal(t, "another value", c.env["anotherkey"])

	c.AddEnv("key", "newvalue")
	assert.Equal(t, "newvalue", c.env["key"])

	assert.Len(t, c.env, 2)
}

func TestRemoveEnv(t *testing.T) {
	c := Container{env: map[string]string{
		"key":        "value",
		"anotherkey": "another value",
	}}

	assert.Len(t, c.env, 2)

	c.RemoveEnv("another key")
	assert.Len(t, c.env, 1)

	c.RemoveEnv("   key")
	assert.Empty(t, c.env)
}
