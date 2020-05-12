package bitrise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/hashicorp/go-retryablehttp"
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

// Environment ...
type Environment struct {
	MappedTo string `json:"mapped_to"`
	Value    string `json:"value"`
}

// App ...
type App struct {
	BaseURL           string
	Slug, AccessToken string
	IsDebug           bool
}

// NewAppWithDefaultURL returns a Bitrise client with the default URl
func NewAppWithDefaultURL(slug, accessToken string) App {
	return App{
		BaseURL:     "https://api.bitrise.io",
		Slug:        slug,
		AccessToken: accessToken,
	}
}

// RetryLogAdaptor adapts the retryablehttp.Logger interface to the go-utils logger.
type RetryLogAdaptor struct{}

// Printf implements the retryablehttp.Logger interface
func (*RetryLogAdaptor) Printf(fmtStr string, vars ...interface{}) {
	switch {
	case strings.HasPrefix(fmtStr, "[DEBUG]"):
		log.Printf(strings.TrimSpace(fmtStr[7:]), vars...)
	case strings.HasPrefix(fmtStr, "[ERR]"):
		log.Errorf(strings.TrimSpace(fmtStr[5:]), vars...)
	case strings.HasPrefix(fmtStr, "[ERROR]"):
		log.Errorf(strings.TrimSpace(fmtStr[7:]), vars...)
	case strings.HasPrefix(fmtStr, "[WARN]"):
		log.Warnf(strings.TrimSpace(fmtStr[6:]), vars...)
	case strings.HasPrefix(fmtStr, "[INFO]"):
		log.Infof(strings.TrimSpace(fmtStr[6:]), vars...)
	default:
		log.Printf(fmtStr, vars...)
	}
}

// NewRetryableClient returns a retryable HTTP client
// isDebug sets the timeouts shoreter for testing purposes
func NewRetryableClient(isDebug bool) *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.CheckRetry = retryablehttp.DefaultRetryPolicy
	client.Backoff = retryablehttp.DefaultBackoff
	client.Logger = &RetryLogAdaptor{}
	client.ErrorHandler = retryablehttp.PassthroughErrorHandler
	if !isDebug {
		client.RetryWaitMin = 10 * time.Second
		client.RetryWaitMax = 60 * time.Second
		client.RetryMax = 5
	} else {
		client.RetryWaitMin = 100 * time.Millisecond
		client.RetryWaitMax = 400 * time.Millisecond
		client.RetryMax = 3
	}

	return client
}

// GetBuild ...
func (app App) GetBuild(buildSlug string) (build Build, err error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v0.1/apps/%s/builds/%s", app.BaseURL, app.Slug, buildSlug), nil)
	if err != nil {
		return Build{}, err
	}

	req.Header.Add("Authorization", "token "+app.AccessToken)

	retryReq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return Build{}, fmt.Errorf("failed to create retryable request: %s", err)
	}

	client := NewRetryableClient(app.IsDebug)

	resp, err := client.Do(retryReq)
	if err != nil {
		return Build{}, err
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Build{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Build{}, fmt.Errorf("failed to get response, statuscode: %d, body: %s", resp.StatusCode, respBody)
	}

	var buildResponse buildResponse
	if err := json.Unmarshal(respBody, &buildResponse); err != nil {
		return Build{}, fmt.Errorf("failed to decode response, body: %s, error: %s", respBody, err)
	}
	return buildResponse.Data, nil
}

// StartBuild ...
func (app App) StartBuild(workflow string, buildParams json.RawMessage, buildNumber string, environments []Environment) (startResponse StartResponse, err error) {
	var params map[string]interface{}
	if err := json.Unmarshal(buildParams, &params); err != nil {
		return StartResponse{}, err
	}
	params["workflow_id"] = workflow
	params["skip_git_status_report"] = true

	sourceBuildNumber := Environment{
		MappedTo: "SOURCE_BITRISE_BUILD_NUMBER",
		Value:    buildNumber,
	}

	envs := []Environment{sourceBuildNumber}
	params["environments"] = append(envs, environments...)

	b, err := json.Marshal(params)
	if err != nil {
		return StartResponse{}, nil
	}

	rm := startRequest{HookInfo: hookInfo{Type: "bitrise"}, BuildParams: b}
	b, err = json.Marshal(rm)
	if err != nil {
		return StartResponse{}, nil
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v0.1/apps/%s/builds", app.BaseURL, app.Slug), bytes.NewReader(b))
	if err != nil {
		return StartResponse{}, nil
	}
	req.Header.Add("Authorization", "token "+app.AccessToken)

	retryReq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return StartResponse{}, fmt.Errorf("failed to create retryable request: %s", err)
	}

	retryClient := NewRetryableClient(app.IsDebug)

	resp, err := retryClient.Do(retryReq)
	if err != nil {
		return StartResponse{}, nil
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

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
