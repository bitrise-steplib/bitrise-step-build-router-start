package main

import (
	"reflect"
	"testing"

	"github.com/bitrise-steplib/bitrise-step-build-router-start/bitrise"
)

func Test_createEnvs(t *testing.T) {
	tests := []struct {
		name            string
		environmentKeys string
		want            []bitrise.Environment
	}{
		{
			name:            "empty",
			environmentKeys: "",
			want:            nil,
		},
		{
			name:            "one env",
			environmentKeys: "ENV_1",
			want:            []bitrise.Environment{bitrise.Environment{MappedTo: "ENV_1", Value: ""}},
		},
		{
			name:            "multiple env",
			environmentKeys: "ENV_1\nENV_2\nENV_3\nENV_4",
			want: []bitrise.Environment{
				bitrise.Environment{
					MappedTo: "ENV_1",
					Value:    "",
				},
				bitrise.Environment{
					MappedTo: "ENV_2",
					Value:    "",
				},
				bitrise.Environment{
					MappedTo: "ENV_3",
					Value:    "",
				},
				bitrise.Environment{
					MappedTo: "ENV_4",
					Value:    "",
				},
			},
		},
		{
			name:            "multiple env with $",
			environmentKeys: "ENV_1\n$ENV_2\nENV_3\n$ENV_4",
			want: []bitrise.Environment{
				bitrise.Environment{
					MappedTo: "ENV_1",
					Value:    "",
				},
				bitrise.Environment{
					MappedTo: "ENV_2",
					Value:    "",
				},
				bitrise.Environment{
					MappedTo: "ENV_3",
					Value:    "",
				},
				bitrise.Environment{
					MappedTo: "ENV_4",
					Value:    "",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createEnvs(tt.environmentKeys); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createEnvs() = %v, want %v", got, tt.want)
			}
		})
	}
}
