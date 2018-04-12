package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/bitrise-tools/go-steputils/tools"
	"github.com/trapacska/bitrise-step-build-router-start/bitrise"
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

	var buildSlugs []string

	log.Infof("Starting builds:")

	for _, wf := range strings.Split(cfg.Workflows, "\n") {
		startedBuild, err := app.StartBuild(wf, build.OriginalBuildParams, cfg.BuildNumber)
		if err != nil {
			failf("Failed to start build, error: %s", err)
		}
		buildSlugs = append(buildSlugs, startedBuild.BuildSlug)
		log.Donef("- %s(%s) started", startedBuild.BuildSlug, startedBuild.TriggeredWorkflow)
	}

	if err := tools.ExportEnvironmentWithEnvman(envBuildSlugs, strings.Join(buildSlugs, "\n")); err != nil {
		failf("Failed to export environment variable, error: %s", err)
	}

	if cfg.WaitForBuilds != "true" {
		return
	}

	fmt.Println()
	log.Infof("Waiting for builds:")

	failed := false
	for {
		ct := 0
		for i, buildSlug := range buildSlugs {
			build, err := app.GetBuild(buildSlug)
			if err != nil {
				failf("failed to get build info, error: %s", err)
			}
			if build.Status == 0 {
				ct++
			} else {
				switch build.Status {
				case 1:
					log.Donef("- %s(%s) successful", build.Slug, build.TriggeredWorkflow)
					break
				case 2:
					failed = true
					log.Errorf("- %s(%s) failed", build.Slug, build.TriggeredWorkflow)
					break
				case 3:
					log.Warnf("- %s(%s) aborted", build.Slug, build.TriggeredWorkflow)
					break
				}

				if len(buildSlugs) > 0 {
					buildSlugs = append(buildSlugs[:i], buildSlugs[i+1:]...)
				}
			}
		}
		if ct == 0 {
			log.Donef("All builds finished")
			break
		}

		time.Sleep(time.Second * 3)
	}

	if failed {
		os.Exit(1)
	}
}
