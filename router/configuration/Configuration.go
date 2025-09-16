package configuration

import (
	"maps"
	"os"
	"strings"
	"sync"

	"github.com/Rafael24595/go-web/router/utils"
)

var (
	instance *Configuration
	once     sync.Once
)

// Configuration holds global application settings.
type Configuration struct {
	dev          bool
	traceRequest bool
}

// Instance returns the singleton instance of Configuration.
// The instance is initialized only once by reading environment variables
// from the ".env" file. Expected variables are:
//
//   - GO_WEB_DEV: enables or disables development mode.
//   - GO_WEB_TRACE_REQUEST: enables or disables HTTP request tracing.
//
// If these environment variables are not present, default values (false) are used.
func Instance() Configuration {
	once.Do(func() {
		kargs := readAllEnv(".env")

		instance = &Configuration{
			dev:          kargs["GO_WEB_DEV"].Boold(false),
			traceRequest: kargs["GO_WEB_TRACE_REQUEST"].Boold(false),
		}
	})

	return *instance
}

// Dev reports whether the application is running in development mode.
func (c Configuration) Dev() bool {
	return c.dev
}

// TraceRequest reports whether HTTP request tracing is enabled.
func (c Configuration) TraceRequest() bool {
	return c.traceRequest
}

func readAllEnv(path string) map[string]utils.Argument {
	envs := readDotEnv(path)
	maps.Copy(envs, readEnv())
	return envs
}

func readDotEnv(path string) map[string]utils.Argument {
	envs := make(map[string]utils.Argument)

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return envs
	}

	result, err := os.ReadFile(path)
	if err != nil {
		return envs
	}

	for line := range strings.SplitSeq(string(result), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if key, value, ok := manageEnv(line); ok {
			envs[key] = *value
		}
	}

	return envs
}

func readEnv() map[string]utils.Argument {
	envs := make(map[string]utils.Argument)
	for _, env := range os.Environ() {
		if key, value, ok := manageEnv(env); ok {
			envs[key] = *value
		}
	}
	return envs
}

func manageEnv(env string) (string, *utils.Argument, bool) {
	parts := strings.SplitN(env, "=", 2)
	if len(parts) == 2 {
		return parts[0], utils.ArgumentFrom(parts[1]), true
	}
	return "", nil, false
}
