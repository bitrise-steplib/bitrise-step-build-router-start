package bitrise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// App ...
type App struct {
	Slug, AccessToken string
}

// NewApp ...
func NewApp(appSlug, accessToken string) App {
	return App{
		Slug:        appSlug,
		AccessToken: accessToken,
	}
}

// GetBuild ...
func (app App) GetBuild(buildSlug string) (Build, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds/%s", app.Slug, buildSlug), nil)
	if err != nil {
		return Build{}, err
	}

	req.Header.Add("Authorization", "token "+app.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Build{}, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Build{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Build{}, fmt.Errorf("failed to get response, statuscode: %d, body: %s", resp.StatusCode, respBody)
	}

	var build buildResponse
	err = json.Unmarshal(respBody, &build)
	if err != nil {
		return Build{}, fmt.Errorf("failed to decode response, body: %s, error: %s", respBody, err)
	}
	return build.Data, nil
}

// StartBuild ...
func (app App) StartBuild(workflow string, buildParams json.RawMessage, buildNumber string) (StartResponse, error) {
	var bParams map[string]interface{}
	err := json.Unmarshal(buildParams, &bParams)
	if err != nil {
		return StartResponse{}, err
	}
	bParams["workflow_id"] = workflow
	bParams["environments"] = append(bParams["environments"].([]interface{}),
		map[string]interface{}{
			"is_expand": true,
			"mapped_to": "SOURCE_BITRISE_BUILD_NUMBER",
			"value":     buildNumber,
		})

	buildParams, err = json.Marshal(bParams)
	if err != nil {
		return StartResponse{}, nil
	}

	rm := StartRequest{HookInfo: HookInfo{Type: "bitrise"}, BuildParams: buildParams}
	bodyJSON, err := json.Marshal(rm)
	if err != nil {
		return StartResponse{}, nil
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds", app.Slug), bytes.NewReader(bodyJSON))
	if err != nil {
		return StartResponse{}, nil
	}
	req.Header.Add("Authorization", "token "+app.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return StartResponse{}, nil
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return StartResponse{}, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return StartResponse{}, fmt.Errorf("failed to get response, statuscode: %d, body: %s", resp.StatusCode, respBody)
	}

	var response StartResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return StartResponse{}, fmt.Errorf("failed to decode response, body: %s, error: %s", respBody, err)
	}
	return response, nil
}

func remove(slice []string, what string) (b []string) {
	for _, s := range slice {
		if s != what {
			b = append(b, s)
		}
	}
	return
}

// WaitForBuilds ...
func (app App) WaitForBuilds(buildSlugs []string, statusChangeCallback func(build Build)) error {
	failed := false
	for {
		running := 0
		for _, buildSlug := range buildSlugs {
			build, err := app.GetBuild(buildSlug)
			if err != nil {
				return fmt.Errorf("failed to get build info, error: %s", err)
			}

			if build.Status == 0 {
				running++
				continue
			}

			failed = build.Status != 1

			statusChangeCallback(build)

			buildSlugs = remove(buildSlugs, buildSlug)
		}
		if running == 0 {
			break
		}
		time.Sleep(time.Second * 3)
	}
	if failed {
		return fmt.Errorf("at least one build failed or aborted")
	}
	return nil
}
