package types

// SlackEventType represents valid Slack event types that can be used with App.Event()
// These constants correspond to the event types defined in Slack's Events API
// Reference: https://api.slack.com/events
type SlackEventType string

const (
	// Message Events
	EventTypeMessage SlackEventType = "message"

	// App Events
	EventTypeAppMention         SlackEventType = "app_mention"
	EventTypeAppHomeOpened      SlackEventType = "app_home_opened"
	EventTypeAppUninstalled     SlackEventType = "app_uninstalled"
	EventTypeAppRateLimited     SlackEventType = "app_rate_limited"
	EventTypeAppRequestedToJoin SlackEventType = "app_requested_to_join"

	// Channel Events
	EventTypeChannelArchive        SlackEventType = "channel_archive"
	EventTypeChannelCreated        SlackEventType = "channel_created"
	EventTypeChannelDeleted        SlackEventType = "channel_deleted"
	EventTypeChannelHistoryChanged SlackEventType = "channel_history_changed"
	EventTypeChannelLeft           SlackEventType = "channel_left"
	EventTypeChannelRename         SlackEventType = "channel_rename"
	EventTypeChannelShared         SlackEventType = "channel_shared"
	EventTypeChannelUnarchive      SlackEventType = "channel_unarchive"
	EventTypeChannelUnshared       SlackEventType = "channel_unshared"

	// Direct Message Events
	EventTypeDndUpdated     SlackEventType = "dnd_updated"
	EventTypeDndUpdatedUser SlackEventType = "dnd_updated_user"

	// Email Events
	EventTypeEmailDomainChanged SlackEventType = "email_domain_changed"

	// Emoji Events
	EventTypeEmojiChanged SlackEventType = "emoji_changed"

	// File Events
	EventTypeFileChange         SlackEventType = "file_change"
	EventTypeFileCommentAdded   SlackEventType = "file_comment_added"
	EventTypeFileCommentDeleted SlackEventType = "file_comment_deleted"
	EventTypeFileCommentEdited  SlackEventType = "file_comment_edited"
	EventTypeFileCreated        SlackEventType = "file_created"
	EventTypeFileDeleted        SlackEventType = "file_deleted"
	EventTypeFilePublic         SlackEventType = "file_public"
	EventTypeFileShared         SlackEventType = "file_shared"
	EventTypeFileUnshared       SlackEventType = "file_unshared"

	// Function Events (Slack Functions)
	EventTypeFunctionExecuted SlackEventType = "function_executed"

	// Grid Events
	EventTypeGridMigrationFinished SlackEventType = "grid_migration_finished"
	EventTypeGridMigrationStarted  SlackEventType = "grid_migration_started"

	// Group Events (Private Channels)
	EventTypeGroupArchive        SlackEventType = "group_archive"
	EventTypeGroupClose          SlackEventType = "group_close"
	EventTypeGroupDeleted        SlackEventType = "group_deleted"
	EventTypeGroupHistoryChanged SlackEventType = "group_history_changed"
	EventTypeGroupLeft           SlackEventType = "group_left"
	EventTypeGroupOpen           SlackEventType = "group_open"
	EventTypeGroupRename         SlackEventType = "group_rename"
	EventTypeGroupUnarchive      SlackEventType = "group_unarchive"

	// IM Events (Direct Messages)
	EventTypeImClose          SlackEventType = "im_close"
	EventTypeImCreated        SlackEventType = "im_created"
	EventTypeImHistoryChanged SlackEventType = "im_history_changed"
	EventTypeImOpen           SlackEventType = "im_open"

	// Invite Events
	EventTypeInviteRequested SlackEventType = "invite_requested"

	// Link Events
	EventTypeLinkShared SlackEventType = "link_shared"

	// Member Events
	EventTypeMemberJoinedChannel SlackEventType = "member_joined_channel"
	EventTypeMemberLeftChannel   SlackEventType = "member_left_channel"

	// Message Metadata Events
	EventTypeMessageMetadataDeleted SlackEventType = "message_metadata_deleted"
	EventTypeMessageMetadataPosted  SlackEventType = "message_metadata_posted"
	EventTypeMessageMetadataUpdated SlackEventType = "message_metadata_updated"

	// Pin Events
	EventTypePinAdded   SlackEventType = "pin_added"
	EventTypePinRemoved SlackEventType = "pin_removed"

	// Reaction Events
	EventTypeReactionAdded   SlackEventType = "reaction_added"
	EventTypeReactionRemoved SlackEventType = "reaction_removed"

	// Resources Events
	EventTypeResourcesAdded   SlackEventType = "resources_added"
	EventTypeResourcesRemoved SlackEventType = "resources_removed"

	// Scope Events
	EventTypeScopeGranted SlackEventType = "scope_granted"
	EventTypeScopeDenied  SlackEventType = "scope_denied"

	// Star Events
	EventTypeStarAdded   SlackEventType = "star_added"
	EventTypeStarRemoved SlackEventType = "star_removed"

	// Subteam Events
	EventTypeSubteamCreated        SlackEventType = "subteam_created"
	EventTypeSubteamMembersChanged SlackEventType = "subteam_members_changed"
	EventTypeSubteamSelfAdded      SlackEventType = "subteam_self_added"
	EventTypeSubteamSelfRemoved    SlackEventType = "subteam_self_removed"
	EventTypeSubteamUpdated        SlackEventType = "subteam_updated"

	// Team Events
	EventTypeTeamAccessGranted SlackEventType = "team_access_granted"
	EventTypeTeamAccessRevoked SlackEventType = "team_access_revoked"
	EventTypeTeamDomainChange  SlackEventType = "team_domain_change"
	EventTypeTeamJoin          SlackEventType = "team_join"
	EventTypeTeamRename        SlackEventType = "team_rename"

	// Tokens Events
	EventTypeTokensRevoked SlackEventType = "tokens_revoked"

	// User Events
	EventTypeUserChange          SlackEventType = "user_change"
	EventTypeUserHuddleChanged   SlackEventType = "user_huddle_changed"
	EventTypeUserProfileChanged  SlackEventType = "user_profile_changed"
	EventTypeUserResourceDenied  SlackEventType = "user_resource_denied"
	EventTypeUserResourceGranted SlackEventType = "user_resource_granted"
	EventTypeUserResourceRemoved SlackEventType = "user_resource_removed"
	EventTypeUserStatusChanged   SlackEventType = "user_status_changed"

	// Workflow Events
	EventTypeWorkflowDeleted     SlackEventType = "workflow_deleted"
	EventTypeWorkflowPublished   SlackEventType = "workflow_published"
	EventTypeWorkflowStepDeleted SlackEventType = "workflow_step_deleted"
	EventTypeWorkflowStepExecute SlackEventType = "workflow_step_execute"
	EventTypeWorkflowUnpublished SlackEventType = "workflow_unpublished"
)

// String returns the string representation of the event type
func (e SlackEventType) String() string {
	return string(e)
}

// IsValid checks if the event type is a valid Slack event type
func (e SlackEventType) IsValid() bool {
	switch e {
	case EventTypeMessage,
		EventTypeAppMention, EventTypeAppHomeOpened, EventTypeAppUninstalled, EventTypeAppRateLimited, EventTypeAppRequestedToJoin,
		EventTypeChannelArchive, EventTypeChannelCreated, EventTypeChannelDeleted, EventTypeChannelHistoryChanged, EventTypeChannelLeft, EventTypeChannelRename, EventTypeChannelShared, EventTypeChannelUnarchive, EventTypeChannelUnshared,
		EventTypeDndUpdated, EventTypeDndUpdatedUser,
		EventTypeEmailDomainChanged,
		EventTypeEmojiChanged,
		EventTypeFileChange, EventTypeFileCommentAdded, EventTypeFileCommentDeleted, EventTypeFileCommentEdited, EventTypeFileCreated, EventTypeFileDeleted, EventTypeFilePublic, EventTypeFileShared, EventTypeFileUnshared,
		EventTypeFunctionExecuted,
		EventTypeGridMigrationFinished, EventTypeGridMigrationStarted,
		EventTypeGroupArchive, EventTypeGroupClose, EventTypeGroupDeleted, EventTypeGroupHistoryChanged, EventTypeGroupLeft, EventTypeGroupOpen, EventTypeGroupRename, EventTypeGroupUnarchive,
		EventTypeImClose, EventTypeImCreated, EventTypeImHistoryChanged, EventTypeImOpen,
		EventTypeInviteRequested,
		EventTypeLinkShared,
		EventTypeMemberJoinedChannel, EventTypeMemberLeftChannel,
		EventTypeMessageMetadataDeleted, EventTypeMessageMetadataPosted, EventTypeMessageMetadataUpdated,
		EventTypePinAdded, EventTypePinRemoved,
		EventTypeReactionAdded, EventTypeReactionRemoved,
		EventTypeResourcesAdded, EventTypeResourcesRemoved,
		EventTypeScopeGranted, EventTypeScopeDenied,
		EventTypeStarAdded, EventTypeStarRemoved,
		EventTypeSubteamCreated, EventTypeSubteamMembersChanged, EventTypeSubteamSelfAdded, EventTypeSubteamSelfRemoved, EventTypeSubteamUpdated,
		EventTypeTeamAccessGranted, EventTypeTeamAccessRevoked, EventTypeTeamDomainChange, EventTypeTeamJoin, EventTypeTeamRename,
		EventTypeTokensRevoked,
		EventTypeUserChange, EventTypeUserHuddleChanged, EventTypeUserProfileChanged, EventTypeUserResourceDenied, EventTypeUserResourceGranted, EventTypeUserResourceRemoved, EventTypeUserStatusChanged,
		EventTypeWorkflowDeleted, EventTypeWorkflowPublished, EventTypeWorkflowStepDeleted, EventTypeWorkflowStepExecute, EventTypeWorkflowUnpublished:
		return true
	default:
		return false
	}
}

// AllEventTypes returns a slice of all valid event types
func AllEventTypes() []SlackEventType {
	return []SlackEventType{
		EventTypeMessage,
		EventTypeAppMention, EventTypeAppHomeOpened, EventTypeAppUninstalled, EventTypeAppRateLimited, EventTypeAppRequestedToJoin,
		EventTypeChannelArchive, EventTypeChannelCreated, EventTypeChannelDeleted, EventTypeChannelHistoryChanged, EventTypeChannelLeft, EventTypeChannelRename, EventTypeChannelShared, EventTypeChannelUnarchive, EventTypeChannelUnshared,
		EventTypeDndUpdated, EventTypeDndUpdatedUser,
		EventTypeEmailDomainChanged,
		EventTypeEmojiChanged,
		EventTypeFileChange, EventTypeFileCommentAdded, EventTypeFileCommentDeleted, EventTypeFileCommentEdited, EventTypeFileCreated, EventTypeFileDeleted, EventTypeFilePublic, EventTypeFileShared, EventTypeFileUnshared,
		EventTypeFunctionExecuted,
		EventTypeGridMigrationFinished, EventTypeGridMigrationStarted,
		EventTypeGroupArchive, EventTypeGroupClose, EventTypeGroupDeleted, EventTypeGroupHistoryChanged, EventTypeGroupLeft, EventTypeGroupOpen, EventTypeGroupRename, EventTypeGroupUnarchive,
		EventTypeImClose, EventTypeImCreated, EventTypeImHistoryChanged, EventTypeImOpen,
		EventTypeInviteRequested,
		EventTypeLinkShared,
		EventTypeMemberJoinedChannel, EventTypeMemberLeftChannel,
		EventTypeMessageMetadataDeleted, EventTypeMessageMetadataPosted, EventTypeMessageMetadataUpdated,
		EventTypePinAdded, EventTypePinRemoved,
		EventTypeReactionAdded, EventTypeReactionRemoved,
		EventTypeResourcesAdded, EventTypeResourcesRemoved,
		EventTypeScopeGranted, EventTypeScopeDenied,
		EventTypeStarAdded, EventTypeStarRemoved,
		EventTypeSubteamCreated, EventTypeSubteamMembersChanged, EventTypeSubteamSelfAdded, EventTypeSubteamSelfRemoved, EventTypeSubteamUpdated,
		EventTypeTeamAccessGranted, EventTypeTeamAccessRevoked, EventTypeTeamDomainChange, EventTypeTeamJoin, EventTypeTeamRename,
		EventTypeTokensRevoked,
		EventTypeUserChange, EventTypeUserHuddleChanged, EventTypeUserProfileChanged, EventTypeUserResourceDenied, EventTypeUserResourceGranted, EventTypeUserResourceRemoved, EventTypeUserStatusChanged,
		EventTypeWorkflowDeleted, EventTypeWorkflowPublished, EventTypeWorkflowStepDeleted, EventTypeWorkflowStepExecute, EventTypeWorkflowUnpublished,
	}
}
