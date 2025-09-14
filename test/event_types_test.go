package test

import (
	"testing"

	"github.com/Asafrose/bolt-go/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestSlackEventType(t *testing.T) {
	t.Run("should have correct string representation", func(t *testing.T) {
		assert.Equal(t, "message", types.EventTypeMessage.String())
		assert.Equal(t, "app_mention", types.EventTypeAppMention.String())
		assert.Equal(t, "message_metadata_posted", types.EventTypeMessageMetadataPosted.String())
		assert.Equal(t, "reaction_added", types.EventTypeReactionAdded.String())
	})

	t.Run("should validate known event types", func(t *testing.T) {
		assert.True(t, types.EventTypeMessage.IsValid())
		assert.True(t, types.EventTypeAppMention.IsValid())
		assert.True(t, types.EventTypeMessageMetadataPosted.IsValid())
		assert.True(t, types.EventTypeReactionAdded.IsValid())
		assert.True(t, types.EventTypeFunctionExecuted.IsValid())
	})

	t.Run("should reject invalid event types", func(t *testing.T) {
		invalidEvent := types.SlackEventType("invalid_event_type")
		assert.False(t, invalidEvent.IsValid())

		anotherInvalid := types.SlackEventType("not_a_real_event")
		assert.False(t, anotherInvalid.IsValid())
	})

	t.Run("should return all event types", func(t *testing.T) {
		allTypes := types.AllEventTypes()
		assert.Greater(t, len(allTypes), 50, "Should have many event types")

		// Check that some key event types are included
		assert.Contains(t, allTypes, types.EventTypeMessage)
		assert.Contains(t, allTypes, types.EventTypeAppMention)
		assert.Contains(t, allTypes, types.EventTypeMessageMetadataPosted)
		assert.Contains(t, allTypes, types.EventTypeReactionAdded)
	})
}
