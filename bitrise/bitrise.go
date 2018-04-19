package bitrise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Build ...
type Build struct {
	Slug                string          `json:"slug"`
	Status              int             `json:"status"`
	StatusText          string          `json:"status_text"`
	BuildNumber         int64           `json:"build_number"`
	TriggeredWorkflow   string          `json:"triggered_workflow"`
	OriginalBuildParams json.RawMessage `json:"original_build_params"`
}

type buildResponse struct {
	Data Build `json:"data"`
}

type hookInfo struct {
	Type string `json:"type"`
}

type startRequest struct {
	HookInfo    hookInfo        `json:"hook_info"`
	BuildParams json.RawMessage `json:"build_params"`
}

// StartResponse ...
type StartResponse struct {
	Status            string `json:"message"`
	Message           string `json:"status"`
	BuildSlug         string `json:"build_slug"`
	BuildNumber       int    `json:"build_number"`
	BuildURL          string `json:"build_url"`
	TriggeredWorkflow string `json:"triggered_workflow"`
}

// App ...
type App struct {
	Slug, AccessToken string
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
	if err := json.Unmarshal(respBody, &build); err != nil {
		return Build{}, fmt.Errorf("failed to decode response, body: %s, error: %s", respBody, err)
	}
	return build.Data, nil
}

// StartBuild ...
func (app App) StartBuild(workflow string, buildParams json.RawMessage, buildNumber string) (StartResponse, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(buildParams, &params); err != nil {
		return StartResponse{}, err
	}
	params["workflow_id"] = workflow
	params["skip_git_status_report"] = true

	sourceBuildNumber := map[string]interface{}{
		"is_expand": true,
		"mapped_to": "SOURCE_BITRISE_BUILD_NUMBER",
		"value":     buildNumber,
	}

	if envs, ok := params["environments"].([]interface{}); ok {
		params["environments"] = append(envs, sourceBuildNumber)
	} else {
		params["environments"] = []interface{}{sourceBuildNumber}
	}

	b, err := json.Marshal(params)
	if err != nil {
		return StartResponse{}, nil
	}

	rm := startRequest{HookInfo: hookInfo{Type: "bitrise"}, BuildParams: b}
	b, err = json.Marshal(rm)
	if err != nil {
		return StartResponse{}, nil
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds", app.Slug), bytes.NewReader(b))
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

// WaitForBuilds ...
func (app App) WaitForBuilds(buildSlugs []string, statusChangeCallback func(build Build)) error {
	failed := false
	status := map[string]string{}
	for {
		running := 0
		for _, buildSlug := range buildSlugs {
			build, err := app.GetBuild(buildSlug)
			if err != nil {
				return fmt.Errorf("failed to get build info, error: %s", err)
			}

			if status[buildSlug] != build.StatusText {
				statusChangeCallback(build)
				status[buildSlug] = build.StatusText
			}

			if build.Status == 0 {
				running++
				continue
			}

			if build.Status != 1 {
				failed = true
			}

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

func remove(slice []string, what string) (b []string) {
	for _, s := range slice {
		if s != what {
			b = append(b, s)
		}
	}
	return
}
