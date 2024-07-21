package dockerutils

import (
	"fmt"
	"strings"
)

type Container struct {
	id  string
	env map[string]string
}

func sanitizeEnvKey(key string) string {
	return strings.TrimSpace(strings.ReplaceAll(key, " ", ""))
}

// returns a list of KEY=VALUE environment variable bindings for the container
func (c *Container) Env() []string {
	env := []string{}
	for key, value := range c.env {
		env = append(env, fmt.Sprintf("%s='%s'", key, value))
	}

	return env
}

// adds and environment variable with KEY and VALUE to container
func (c *Container) AddEnv(key, value string) {
	c.env[sanitizeEnvKey(key)] = value
}

// removes environment variable with KEY if it exists in container
func (c *Container) RemoveEnv(key string) {
	delete(c.env, sanitizeEnvKey(key))
}
