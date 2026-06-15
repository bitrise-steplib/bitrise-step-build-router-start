package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/bitrise-step-build-router-start/bitrise"
)

const (
	envBuildSlugs    = "ROUTER_STARTED_BUILD_SLUGS"
	envPipelineSlugs = "ROUTER_STARTED_PIPELINE_SLUGS"
)

// Config ...
type Config struct {
	AppSlug                string          `env:"BITRISE_APP_SLUG,required"`
	BuildSlug              string          `env:"BITRISE_BUILD_SLUG,required"`
	BuildNumber            string          `env:"BITRISE_BUILD_NUMBER,required"`
	AccessToken            stepconf.Secret `env:"access_token,required"`
	WaitForBuilds          string          `env:"wait_for_builds"`
	BuildArtifactsSavePath string          `env:"build_artifacts_save_path"`
	AbortBuildsOnFail      string          `env:"abort_on_fail"`
	Workflows              string          `env:"workflows"`
	Pipelines              string          `env:"pipelines"`
	Environments           string          `env:"environment_key_list"`
	IsVerboseLog           bool            `env:"verbose,required"`
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

	log.SetEnableDebugLog(cfg.IsVerboseLog)

	workflows := splitLines(cfg.Workflows)
	pipelines := splitLines(cfg.Pipelines)
	if len(workflows) == 0 && len(pipelines) == 0 {
		failf("No workflows or pipelines specified. Set at least one of the 'workflows' or 'pipelines' inputs.")
	}

	app := bitrise.NewAppWithDefaultURL(cfg.AppSlug, string(cfg.AccessToken))

	build, err := app.GetBuild(cfg.BuildSlug)
	if err != nil {
		failf("failed to get build, error: %s", err)
	}

	environments := createEnvs(cfg.Environments)

	var buildSlugs []string
	if len(workflows) > 0 {
		log.Infof("Starting builds:")
		for _, wf := range workflows {
			startedBuild, err := app.StartBuild(wf, build.OriginalBuildParams, cfg.BuildNumber, environments)
			if err != nil {
				failf("Failed to start build, error: %s", err)
			}
			if startedBuild.BuildSlug == "" {
				failf("Build was not started. This could mean that manual build approval is enabled for this project and it's blocking this step from starting builds.")
			}
			buildSlugs = append(buildSlugs, startedBuild.BuildSlug)
			log.Printf("- %s started (https://app.bitrise.io/build/%s)", startedBuild.TriggeredWorkflow, startedBuild.BuildSlug)
		}
	}

	var pipelineSlugs []string
	if len(pipelines) > 0 {
		fmt.Println()
		log.Infof("Starting pipelines:")
		for _, pl := range pipelines {
			startedPipeline, err := app.StartPipeline(pl, build.OriginalBuildParams, cfg.BuildNumber, environments)
			if err != nil {
				failf("Failed to start pipeline, error: %s", err)
			}
			if startedPipeline.Slug == "" {
				failf("Pipeline was not started. This could mean that manual build approval is enabled for this project and it's blocking this step from starting pipelines.")
			}
			pipelineSlugs = append(pipelineSlugs, startedPipeline.Slug)
			log.Printf("- %s started (https://app.bitrise.io/build/%s)", pl, startedPipeline.Slug)
		}
	}

	if err := tools.ExportEnvironmentWithEnvman(envBuildSlugs, strings.Join(buildSlugs, "\n")); err != nil {
		failf("Failed to export environment variable, error: %s", err)
	}
	if err := tools.ExportEnvironmentWithEnvman(envPipelineSlugs, strings.Join(pipelineSlugs, "\n")); err != nil {
		failf("Failed to export environment variable, error: %s", err)
	}

	if cfg.WaitForBuilds != "true" {
		return
	}

	fmt.Println()
	log.Infof("Waiting for builds and pipelines:")

	// abortAll aborts every other started build and pipeline when one of them fails. The slug
	// that triggered the abort is skipped so it doesn't try to abort itself.
	abortAll := func(triggerSlug, failReason string) {
		if cfg.AbortBuildsOnFail != "yes" {
			return
		}
		for _, buildSlug := range buildSlugs {
			if buildSlug == triggerSlug {
				continue
			}
			if abortErr := app.AbortBuild(buildSlug, "Abort on Fail - [https://app.bitrise.io/build/"+triggerSlug+"] "+failReason+"\nAuto aborted by parent build"); abortErr != nil {
				log.Warnf("failed to abort build, error: %s", abortErr)
				continue
			}
			log.Donef("Build " + buildSlug + " aborted due to associated failure")
		}
		for _, pipelineSlug := range pipelineSlugs {
			if pipelineSlug == triggerSlug {
				continue
			}
			if abortErr := app.AbortPipeline(pipelineSlug, "Abort on Fail - [https://app.bitrise.io/build/"+triggerSlug+"] "+failReason+"\nAuto aborted by parent build"); abortErr != nil {
				log.Warnf("failed to abort pipeline, error: %s", abortErr)
				continue
			}
			log.Donef("Pipeline " + pipelineSlug + " aborted due to associated failure")
		}
	}

	buildCallback := func(build bitrise.Build) {
		var failReason string
		switch build.Status {
		case 0:
			log.Printf("- %s %s", build.TriggeredWorkflow, build.StatusText)
		case 1:
			log.Donef("- %s successful", build.TriggeredWorkflow)
		case 2:
			log.Errorf("- %s failed", build.TriggeredWorkflow)
			failReason = "failed"
		case 3:
			log.Warnf("- %s aborted", build.TriggeredWorkflow)
			failReason = "aborted"
		case 4:
			log.Infof("- %s cancelled", build.TriggeredWorkflow)
			failReason = "cancelled"
		}

		if build.Status > 1 {
			abortAll(build.Slug, failReason)
		}

		if build.Status != 0 {
			saveBuildArtifacts(app, build, cfg.BuildArtifactsSavePath)
		}
	}

	pipelineCallback := func(pipeline bitrise.Pipeline) {
		switch {
		case pipeline.IsRunning():
			log.Printf("- %s %s", pipeline.Name, pipeline.Status)
		case pipeline.IsSuccessful():
			log.Donef("- %s %s", pipeline.Name, pipeline.Status)
		case pipeline.IsFailed():
			log.Errorf("- %s failed", pipeline.Name)
			abortAll(pipeline.Slug, "failed")
		case pipeline.IsAborted():
			log.Warnf("- %s aborted", pipeline.Name)
			abortAll(pipeline.Slug, "aborted")
		}
	}

	if err := app.WaitForBuildsAndPipelines(buildSlugs, pipelineSlugs, buildCallback, pipelineCallback); err != nil {
		failf("An error occurred: %s", err)
	}
}

func saveBuildArtifacts(app bitrise.App, build bitrise.Build, savePath string) {
	buildArtifactSaveDir := strings.TrimSpace(savePath)
	if buildArtifactSaveDir == "" {
		return
	}

	artifactsResponse, err := build.GetBuildArtifacts(app)
	if err != nil {
		log.Warnf("failed to get build artifacts: %s", err)
	}
	for _, artifactSlug := range artifactsResponse.ArtifactSlugs {
		artifactObj, err := build.GetBuildArtifact(app, artifactSlug.ArtifactSlug)
		if err != nil {
			log.Warnf("failed to get build artifact: %s", err)
			continue
		}
		if err = os.MkdirAll(buildArtifactSaveDir, 0777); err != nil {
			log.Warnf("failed to ensure artifact path %s exists: %s", buildArtifactSaveDir, err)
			continue
		}
		fullBuildArtifactsSavePath := filepath.Join(buildArtifactSaveDir, artifactObj.Artifact.Title)
		downloadErr := artifactObj.Artifact.DownloadArtifact(fullBuildArtifactsSavePath)
		if downloadErr != nil {
			log.Warnf("failed to download %s artifact: %s", artifactObj.Artifact.Title, downloadErr)
		} else {
			log.Donef("Downloaded %s to %s", artifactObj.Artifact.Title, fullBuildArtifactsSavePath)
		}
	}
}

// splitLines splits a newline separated input into a list of trimmed, non-empty entries.
func splitLines(list string) []string {
	var entries []string
	for _, entry := range strings.Split(strings.TrimSpace(list), "\n") {
		entry = strings.TrimSpace(entry)
		if entry != "" {
			entries = append(entries, entry)
		}
	}
	return entries
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
