package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/bitrise-step-build-router-start/bitrise"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/bitrise-tools/go-steputils/tools"
)

const envBuildSlugs = "ROUTER_STARTED_BUILD_SLUGS"

// Config ...
type Config struct {
	AppSlug       string          `env:"BITRISE_APP_SLUG,required"`
	BuildSlug     string          `env:"BITRISE_BUILD_SLUG,required"`
	BuildNumber   string          `env:"BITRISE_BUILD_NUMBER,required"`
	AccessToken   stepconf.Secret `env:"access_token,required"`
	WaitForBuilds string          `env:"wait_for_builds"`
	Workflows     string          `env:"workflows,required"`
}

func failf(s string, a ...interface{}) {
	log.Errorf(s, a...)
	os.Exit(1)
}

func main() {
	var cfg Config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Issue with an input: %s", err)
	}

	stepconf.Print(cfg)
	fmt.Println()

	app := bitrise.NewApp(cfg.AppSlug, string(cfg.AccessToken))

	build, err := app.GetBuild(cfg.BuildSlug)
	if err != nil {
		failf("failed to get build, error: %s", err)
	}

	log.Infof("Starting builds:")

	var buildSlugs []string
	for _, wf := range strings.Split(cfg.Workflows, "\n") {
		startedBuild, err := app.StartBuild(wf, build.OriginalBuildParams, cfg.BuildNumber)
		if err != nil {
			failf("Failed to start build, error: %s", err)
		}
		buildSlugs = append(buildSlugs, startedBuild.BuildSlug)
		log.Printf("- %s started (https://www.bitrise.io/build/%s)", startedBuild.TriggeredWorkflow, startedBuild.BuildSlug)
	}

	if err := tools.ExportEnvironmentWithEnvman(envBuildSlugs, strings.Join(buildSlugs, "\n")); err != nil {
		failf("Failed to export environment variable, error: %s", err)
	}

	if cfg.WaitForBuilds != "true" {
		return
	}

	fmt.Println()
	log.Infof("Waiting for builds:")

	if err := app.WaitForBuilds(buildSlugs, func(build bitrise.Build) {
		switch build.Status {
		case 1:
			log.Donef("- %s successful (https://www.bitrise.io/build/%s)", build.TriggeredWorkflow, build.Slug)
		case 2:
			log.Errorf("- %s failed (https://www.bitrise.io/build/%s)", build.TriggeredWorkflow, build.Slug)
		case 3:
			log.Warnf("- %s aborted (https://www.bitrise.io/build/%s)", build.TriggeredWorkflow, build.Slug)
		}
	}); err != nil {
		failf("An error occoured: %s", err)
	}
}
