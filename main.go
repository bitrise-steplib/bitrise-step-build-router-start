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
	Environments  string          `env:"environment_key_list"`
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

	app := bitrise.App{
		Slug:        cfg.AppSlug,
		AccessToken: string(cfg.AccessToken),
	}

	build, err := app.GetBuild(cfg.BuildSlug)
	if err != nil {
		failf("failed to get build, error: %s", err)
	}

	log.Infof("Starting builds:")

	var buildSlugs []string
	environments := createEnvs(cfg.Environments)
	for _, wf := range strings.Split(strings.TrimSpace(cfg.Workflows), "\n") {
		wf = strings.TrimSpace(wf)
		startedBuild, err := app.StartBuild(wf, build.OriginalBuildParams, cfg.BuildNumber, environments)
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
		case 0:
			log.Printf("- %s %s", build.TriggeredWorkflow, build.StatusText)
		case 1:
			log.Donef("- %s successful", build.TriggeredWorkflow)
		case 2:
			log.Errorf("- %s failed", build.TriggeredWorkflow)
		case 3:
			log.Warnf("- %s aborted", build.TriggeredWorkflow)
		case 4:
			log.Infof("- %s cancelled", build.TriggeredWorkflow)
		}
	}); err != nil {
		failf("An error occoured: %s", err)
	}
}

func createEnvs(environmentKeys string) []bitrise.Environment {
	environmentKeys = strings.Replace(environmentKeys, "$", "", -1)
	environmentsKeyList := strings.Split(environmentKeys, "\n")

	var environments []bitrise.Environment
	for _, key := range environmentsKeyList {
		if key == "" {
			continue
		}

		env := bitrise.Environment{
			MappedTo: key,
			Value:    os.Getenv(key),
		}
		environments = append(environments, env)
	}
	return environments
}
