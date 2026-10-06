package plugin

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestCommandBuilder(t *testing.T) {
	tests := []struct {
		Name      string
		Daemon    Daemon
		Input     string
		WantedLen int
		Skip      bool
		Excuse    string
	}{
		{
			Name:      "Single driver-opt value",
			Daemon:    Daemon{},
			Input:     "no_proxy=*.mydomain",
			WantedLen: 1,
		},
		{
			Name:      "Single driver-opt value with comma",
			Input:     "no_proxy=.mydomain,.sub.domain.com",
			WantedLen: 1,
			Skip:      true,
			Excuse:    "Can be enabled whenever #94 is fixed.",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if test.Skip {
				t.Skip(fmt.Printf("%v skipped. %v", test.Name, test.Excuse))
			}
			// prepare test values to mock plugin call with settings
			os.Setenv("PLUGIN_BUILDKIT_DRIVEROPT", test.Input)

			// create dummy cli app to reproduce the issue
			app := &cli.App{
				Name:    "dummy App",
				Usage:   "testing inputs",
				Version: "0.0.1",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:        "daemon.buildkit-driveropt",
						EnvVars:     []string{"PLUGIN_BUILDKIT_DRIVEROPT"},
						Usage:       "adds optional driver-ops args like 'env.http_proxy'",
						Destination: &test.Daemon.BuildkitDriverOpt,
					},
				},
				Action: nil,
			}

			// need to run the app to resolve the flags
			_ = app.Run(nil)

			// call the commandBuilder to prepare the cmd with its args
			_ = commandBuilder(test.Daemon)

			assert.Len(t, test.Daemon.BuildkitDriverOpt.Value(), test.WantedLen)
		})
	}
}

func TestCommandBuildCacheKeyInputs(t *testing.T) {
	build := Build{Dockerfile: "Dockerfile", Context: "."}

	args := strings.Join(commandBuild(build, false).Args, " ")
	assert.NotContains(t, args, "SOURCE_DATE_EPOCH")
	assert.Contains(t, args, "--output type=image,push=true")
	assert.NotContains(t, args, "rewrite-timestamp")

	args = strings.Join(commandBuild(build, true).Args, " ")
	assert.Contains(t, args, "--output type=image")
	assert.NotContains(t, args, "push=true")

	build.Reproducible = true
	args = strings.Join(commandBuild(build, false).Args, " ")
	assert.Contains(t, args, "--output type=image,push=true,rewrite-timestamp=true")
}

func TestAutoCacheImage(t *testing.T) {
	assert.Equal(t, "org/app:buildcache", autoCacheImage("org/app", ""))
	assert.Equal(t, "org/app:buildcache-web", autoCacheImage("org/app", "web"))
}
