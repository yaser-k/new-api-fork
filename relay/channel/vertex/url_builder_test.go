package vertex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildGoogleModelURLUsesTheLocationHost(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"global", "https://aiplatform.googleapis.com/v1/projects/my-project/locations/global/publishers/google/models/gemini-4-argon:generateContent"},
		{"us-central1", "https://us-central1-aiplatform.googleapis.com/v1/projects/my-project/locations/us-central1/publishers/google/models/gemini-4-argon:generateContent"},
		{"us", "https://aiplatform.us.rep.googleapis.com/v1/projects/my-project/locations/us/publishers/google/models/gemini-4-argon:generateContent"},
		{"eu", "https://aiplatform.eu.rep.googleapis.com/v1/projects/my-project/locations/eu/publishers/google/models/gemini-4-argon:generateContent"},
	}
	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			got := BuildGoogleModelURL("", DefaultAPIVersion, "my-project", tt.region, "gemini-4-argon", "generateContent")
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildMistralModelURLUsesTheMistralPublisher(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{"rawPredict", "https://europe-west4-aiplatform.googleapis.com/v1/projects/my-project/locations/europe-west4/publishers/mistralai/models/mistral-medium-3:rawPredict"},
		{"streamRawPredict", "https://europe-west4-aiplatform.googleapis.com/v1/projects/my-project/locations/europe-west4/publishers/mistralai/models/mistral-medium-3:streamRawPredict"},
	}
	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			got := BuildMistralModelURL("", DefaultAPIVersion, "my-project", "europe-west4", "mistral-medium-3", tt.action)
			assert.Equal(t, tt.want, got)
		})
	}
}
