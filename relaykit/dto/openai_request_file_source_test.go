package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMediaContentInputAudioFileSourceMimeType(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		wantMime string
	}{
		{name: "bare mp3", format: "mp3", wantMime: "audio/mp3"},
		{name: "bare wav", format: "wav", wantMime: "audio/wav"},
		{name: "full mime type", format: "audio/mpeg", wantMime: "audio/mpeg"},
		{name: "empty format", format: "", wantMime: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := MediaContent{
				Type:       ContentTypeInputAudio,
				InputAudio: &MessageInputAudio{Data: "UklGRg==", Format: tt.format},
			}

			source, ok := content.ToFileSource().(*types.Base64Source)
			require.True(t, ok)
			assert.Equal(t, tt.wantMime, source.MimeType)
		})
	}
}
