package test

import (
	"encoding/json"
	"testing"

	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseType(t *testing.T) {
	t.Parallel()

	t.Run("should have correct string values", func(t *testing.T) {
		assert.Equal(t, "ephemeral", types.ResponseTypeEphemeral.String())
		assert.Equal(t, "in_channel", types.ResponseTypeInChannel.String())
	})

	t.Run("should validate correctly", func(t *testing.T) {
		assert.True(t, types.ResponseTypeEphemeral.IsValid())
		assert.True(t, types.ResponseTypeInChannel.IsValid())
		assert.False(t, types.ResponseType("invalid").IsValid())
	})

	t.Run("should return all valid types", func(t *testing.T) {
		allTypes := types.AllResponseTypes()
		assert.Len(t, allTypes, 2)
		assert.Contains(t, allTypes, types.ResponseTypeEphemeral)
		assert.Contains(t, allTypes, types.ResponseTypeInChannel)
	})

	t.Run("should marshal to correct JSON values", func(t *testing.T) {
		// Test RespondArguments marshaling
		respondArgs := types.RespondArguments{
			Text:         "Test message",
			ResponseType: types.ResponseTypeEphemeral,
		}

		data, err := json.Marshal(respondArgs)
		require.NoError(t, err)

		// Should contain the string value, not the type name
		assert.Contains(t, string(data), `"response_type":"ephemeral"`)
		assert.Contains(t, string(data), `"text":"Test message"`)
	})

	t.Run("should unmarshal from JSON correctly", func(t *testing.T) {
		jsonData := `{"text":"Test message","response_type":"ephemeral"}`

		var respondArgs types.RespondArguments
		err := json.Unmarshal([]byte(jsonData), &respondArgs)
		require.NoError(t, err)

		assert.Equal(t, "Test message", respondArgs.Text)
		assert.Equal(t, types.ResponseTypeEphemeral, respondArgs.ResponseType)
	})

	t.Run("should work with CommandResponse", func(t *testing.T) {
		// Test CommandResponse marshaling
		commandResp := types.CommandResponse{
			Text:         "Command response",
			ResponseType: types.ResponseTypeInChannel,
			Blocks: []slack.Block{
				&slack.SectionBlock{
					Type: slack.MBTSection,
					Text: &slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: "*Bold text*",
					},
				},
			},
		}

		data, err := json.Marshal(commandResp)
		require.NoError(t, err)

		// Should contain the string value
		assert.Contains(t, string(data), `"response_type":"in_channel"`)
		assert.Contains(t, string(data), `"text":"Command response"`)
	})

	t.Run("should handle empty response type", func(t *testing.T) {
		respondArgs := types.RespondArguments{
			Text: "Test message",
			// ResponseType is zero value (empty string)
		}

		data, err := json.Marshal(respondArgs)
		require.NoError(t, err)

		// Empty ResponseType should not appear in JSON due to omitempty
		assert.NotContains(t, string(data), `"response_type"`)
		assert.Contains(t, string(data), `"text":"Test message"`)
	})

	t.Run("should handle unmarshaling invalid response type", func(t *testing.T) {
		jsonData := `{"text":"Test message","response_type":"invalid_type"}`

		var respondArgs types.RespondArguments
		err := json.Unmarshal([]byte(jsonData), &respondArgs)
		require.NoError(t, err) // JSON unmarshaling should succeed

		assert.Equal(t, "Test message", respondArgs.Text)
		assert.Equal(t, types.ResponseType("invalid_type"), respondArgs.ResponseType)
		assert.False(t, respondArgs.ResponseType.IsValid()) // But validation should fail
	})
}
