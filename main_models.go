package main

import (
	"encoding/json"
	"time"
)

// BuildResponseItemModel ...
type BuildResponseItemModel struct {
	Data struct {
		TriggeredAt                  time.Time       `json:"triggered_at"`
		StartedOnWorkerAt            time.Time       `json:"started_on_worker_at"`
		EnvironmentPrepareFinishedAt time.Time       `json:"environment_prepare_finished_at"`
		FinishedAt                   time.Time       `json:"finished_at"`
		Slug                         string          `json:"slug"`
		Status                       int             `json:"status"`
		StatusText                   string          `json:"status_text"`
		AbortReason                  string          `json:"abort_reason"`
		IsOnHold                     bool            `json:"is_on_hold"`
		Branch                       string          `json:"branch"`
		BuildNumber                  int64           `json:"build_number"`
		CommitHash                   string          `json:"commit_hash"`
		CommitMessage                string          `json:"commit_message"`
		Tag                          string          `json:"tag"`
		TriggeredWorkflow            string          `json:"triggered_workflow"`
		TriggeredBy                  string          `json:"triggered_by"`
		StackConfigType              string          `json:"stack_config_type"`
		StackIdentifier              string          `json:"stack_identifier"`
		OriginalBuildParams          json.RawMessage `json:"original_build_params"`
		PullRequestID                int64           `json:"pull_request_id"`
		PullRequestTargetBranch      string          `json:"pull_request_target_branch"`
		PullRequestViewURL           string          `json:"pull_request_view_url"`
		CommitViewURL                string          `json:"commit_view_url"`
	} `json:"data"`
}

// RequestModel ...
type RequestModel struct {
	HookInfo    HookInfoModel   `json:"hook_info"`
	BuildParams json.RawMessage `json:"build_params"`
}

// HookInfoModel ...
type HookInfoModel struct {
	Type     string `json:"type"`
	APIToken string `json:"api_token"`
}

// ResponseModel ...
type ResponseModel struct {
	Status            string `json:"message"`
	Message           string `json:"status"`
	BuildSlug         string `json:"build_slug"`
	BuildNumber       int    `json:"build_number"`
	BuildURL          string `json:"build_url"`
	TriggeredWorkflow string `json:"triggered_workflow"`
}
