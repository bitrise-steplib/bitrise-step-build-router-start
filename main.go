package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
)

func startBuild(appSlug, accessToken, workflow string, buildParams json.RawMessage) (response ResponseModel, err error) {
	var ed map[string]interface{}
	err = json.Unmarshal(buildParams, &ed)
	if err != nil {
		return
	}
	ed["workflow_id"] = workflow

	edB, err := json.Marshal(ed)
	if err != nil {
		return
	}

	rm := RequestModel{HookInfo: HookInfoModel{Type: "bitrise"}, BuildParams: edB}
	bJSON, err := json.Marshal(rm)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds", appSlug), bytes.NewReader(bJSON))
	if err != nil {
		return
	}
	req.Header.Add("Authorization", "token "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return response, fmt.Errorf("failed to get response, statuscode: %d, body: %s", resp.StatusCode, respBody)
	}

	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return response, fmt.Errorf("failed to decode response, body: %s, error: %s", respBody, err)
	}
	return
}

func getBuildInfo(appSlug, buildSlug, accessToken string) (build BuildResponseItemModel, err error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds/%s", appSlug, buildSlug), nil)
	if err != nil {
		return
	}

	req.Header.Add("Authorization", "token "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return build, fmt.Errorf("failed to get response, statuscode: %d, body: %s", resp.StatusCode, respBody)
	}

	err = json.Unmarshal(respBody, &build)
	if err != nil {
		return build, fmt.Errorf("failed to decode response, body: %s, error: %s", respBody, err)
	}
	return
}

func failf(s string, a ...interface{}) {
	log.Errorf(s, a...)
	os.Exit(1)
}

func main() {
	// get current build infos
	appSlug := os.Getenv("BITRISE_APP_SLUG")
	buildSlug := os.Getenv("BITRISE_BUILD_SLUG")
	accessToken := os.Getenv("BITRISE_ACCESS_TOKEN")

	build, err := getBuildInfo(appSlug, buildSlug, accessToken)
	if err != nil {
		failf("failed to get build, error: %s", err)
	}

	builds := map[string]BuildResponseItemModel{}

	log.Infof("Starting builds:")

	for _, wf := range strings.Split(os.Getenv("workflows"), "\n") {
		startedBuild, err := startBuild(appSlug, accessToken, wf, build.Data.OriginalBuildParams)
		if err != nil {
			failf("failed to start build, error: %s", err)
		}
		builds[startedBuild.BuildSlug] = BuildResponseItemModel{}
		log.Donef("- %s(%s) started", startedBuild.BuildSlug, startedBuild.TriggeredWorkflow)
	}

	failed := false

	log.Infof("Waiting for builds:")
	for {
		ct := 0
		for bSlug := range builds {
			builds[bSlug], err = getBuildInfo(appSlug, bSlug, accessToken)
			if err != nil {
				failf("failed to get build info, error: %s", err)
			}
			if builds[bSlug].Data.Status == 0 {
				ct++
			} else {
				switch builds[bSlug].Data.Status {
				case 1:
					log.Donef("- %s(%s) successful", builds[bSlug].Data.Slug, builds[bSlug].Data.TriggeredWorkflow)
					break
				case 2:
					failed = true
					log.Errorf("- %s(%s) failed", builds[bSlug].Data.Slug, builds[bSlug].Data.TriggeredWorkflow)
					break
				case 3:
					log.Warnf("- %s(%s) aborted", builds[bSlug].Data.Slug, builds[bSlug].Data.TriggeredWorkflow)
					break
				}
				delete(builds, bSlug)
			}
		}
		if ct == 0 {
			log.Donef("all builds finished")
			break
		}

		time.Sleep(time.Second * 3)
	}

	if failed {
		os.Exit(1)
	}
}
