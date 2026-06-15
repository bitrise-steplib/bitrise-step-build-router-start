package bitrise

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApp_GetBuild(t *testing.T) {
	entryCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		entryCount++
		writer.WriteHeader(500)
	}))

	tests := []struct {
		name      string
		app       App
		buildSlug string
		want      Build
		wantErr   bool
	}{
		{
			name: "Retry test",
			app: App{
				BaseURL:             server.URL,
				Slug:                "aaa",
				AccessToken:         "bbb",
				IsDebugRetryTimings: true,
			},
			buildSlug: "ccc",
			want:      Build{},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entryCount = 0
			got, err := tt.app.GetBuild(tt.buildSlug)

			require.Greater(t, entryCount, 2)
			if !tt.wantErr {
				require.NoError(t, err, "App.GetBuild() err")
			} else {
				require.Error(t, err, "App.GetBuild() expected to return error")
			}
			require.Equal(t, tt.want, got, "App.GetBuild()")
		})
	}
}

func TestPipeline_StatusHelpers(t *testing.T) {
	tests := []struct {
		status     string
		running    bool
		successful bool
		failed     bool
		aborted    bool
	}{
		{status: "on_hold", running: true},
		{status: "running", running: true},
		{status: "", running: true},
		{status: "succeeded", successful: true},
		{status: "succeeded_with_abort", successful: true},
		{status: "failed", failed: true},
		{status: "aborted", aborted: true},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			p := Pipeline{Status: tt.status}
			require.Equal(t, tt.running, p.IsRunning(), "IsRunning")
			require.Equal(t, tt.successful, p.IsSuccessful(), "IsSuccessful")
			require.Equal(t, tt.failed, p.IsFailed(), "IsFailed")
			require.Equal(t, tt.aborted, p.IsAborted(), "IsAborted")
		})
	}
}

func TestApp_GetPipeline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/v0.1/apps/app-slug/pipelines/pipeline-slug", req.URL.Path)
		require.Equal(t, "token access-token", req.Header.Get("Authorization"))
		_, _ = writer.Write([]byte(`{"data":{"slug":"pipeline-slug","status":"running","name":"my-pipeline"}}`))
	}))
	defer server.Close()

	app := App{BaseURL: server.URL, Slug: "app-slug", AccessToken: "access-token", IsDebugRetryTimings: true}

	got, err := app.GetPipeline("pipeline-slug")
	require.NoError(t, err)
	require.Equal(t, Pipeline{Slug: "pipeline-slug", Status: "running", Name: "my-pipeline"}, got)
}

func TestApp_StartPipeline_SetsPipelineID(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		var sr startRequest
		require.NoError(t, json.NewDecoder(req.Body).Decode(&sr))
		require.NoError(t, json.Unmarshal(sr.BuildParams, &captured))
		_, _ = writer.Write([]byte(`{"status":"ok","slug":"new-pipeline-slug","triggered_pipeline":"my-pipeline"}`))
	}))
	defer server.Close()

	app := App{BaseURL: server.URL, Slug: "app-slug", AccessToken: "access-token", IsDebugRetryTimings: true}

	// Original params carry a workflow_id which must be cleared when triggering a pipeline.
	got, err := app.StartPipeline("my-pipeline", json.RawMessage(`{"workflow_id":"primary","branch":"main"}`), "42", nil)
	require.NoError(t, err)
	require.Equal(t, "new-pipeline-slug", got.Slug)

	require.Equal(t, "my-pipeline", captured["pipeline_id"])
	require.NotContains(t, captured, "workflow_id")
	require.Equal(t, "main", captured["branch"])
}
