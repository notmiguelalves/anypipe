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

func (c *Container) Env() []string {
	env := []string{}
	for key, value := range c.env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

func (c *Container) AddEnv(key, value string) {
	c.env[sanitizeEnvKey(key)] = value
}

func (c *Container) RemoveEnv(key string) {
	delete(c.env, sanitizeEnvKey(key))
}
