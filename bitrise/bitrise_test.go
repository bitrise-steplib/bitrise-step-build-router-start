package bitrise

import (
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
				BaseURL:     server.URL,
				Slug:        "aaa",
				AccessToken: "bbb",
				IsDebug:     true,
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
