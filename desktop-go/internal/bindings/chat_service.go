package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/chat"
)

type ChatService struct {
	app *app.App
}

func NewChatService(app *app.App) *ChatService {
	return &ChatService{app: app}
}

func (s *ChatService) CreateSession(mode string, title *string, metadata map[string]any, groupID *string) (chat.Session, error) {
	return s.app.Chat.CreateSession(mode, title, metadata, groupID)
}

func (s *ChatService) GetSession(sessionID string) (*chat.Session, error) {
	return s.app.Chat.GetSession(sessionID)
}

func (s *ChatService) LoadSession(sessionID string) (chat.LoadSessionResponse, error) {
	return s.app.Chat.LoadSession(sessionID)
}

func (s *ChatService) SaveSession(sessionID string, sessionState chat.SessionState) (bool, error) {
	return s.app.Chat.SaveSession(sessionID, sessionState)
}

func (s *ChatService) UpdateSessionSettings(sessionID string, settings chat.SessionSettings) (chat.Session, error) {
	return s.app.Chat.UpdateSessionSettings(sessionID, settings)
}

func (s *ChatService) ArchiveSession(sessionID string) (bool, error) {
	return s.app.Chat.ArchiveSession(sessionID)
}

func (s *ChatService) RestoreSession(sessionID string) (chat.Session, error) {
	return s.app.Chat.RestoreSession(sessionID)
}

func (s *ChatService) DeleteSession(sessionID string) (bool, error) {
	return s.app.Chat.DeleteSession(sessionID)
}

func (s *ChatService) MoveSessionToGroup(sessionID string, groupID *string) (bool, error) {
	return s.app.Chat.MoveSessionToGroup(sessionID, groupID)
}

func (s *ChatService) ListSessions(status *string, groupID *string, limit int, offset int) ([]chat.Session, error) {
	return s.app.Chat.ListSessions(status, groupID, limit, offset)
}

func (s *ChatService) CountSessions(status *string, groupID *string) (int, error) {
	return s.app.Chat.CountSessions(status, groupID)
}

func (s *ChatService) CreateGroup(request chat.CreateGroupRequest) (chat.Group, error) {
	return s.app.Chat.CreateGroup(request)
}

func (s *ChatService) GetGroup(groupID string) (*chat.Group, error) {
	return s.app.Chat.GetGroup(groupID)
}

func (s *ChatService) UpdateGroup(groupID string, request chat.UpdateGroupRequest) (chat.Group, error) {
	return s.app.Chat.UpdateGroup(groupID, request)
}

func (s *ChatService) DeleteGroup(groupID string) (bool, error) {
	return s.app.Chat.DeleteGroup(groupID)
}

func (s *ChatService) RestoreGroup(groupID string) (chat.Group, error) {
	return s.app.Chat.RestoreGroup(groupID)
}

func (s *ChatService) ListGroups(status *string, workspaceID *string) ([]chat.Group, error) {
	return s.app.Chat.ListGroups(status, workspaceID)
}

func (s *ChatService) ReorderGroups(groupIDs []string) (bool, error) {
	return s.app.Chat.ReorderGroups(groupIDs)
}

func (s *ChatService) GetSessionTags(sessionID string) ([]string, error) {
	return s.app.Chat.GetSessionTags(sessionID)
}

func (s *ChatService) GetTagsBatch(sessionIDs []string) (map[string][]string, error) {
	return s.app.Chat.GetTagsBatch(sessionIDs)
}

func (s *ChatService) AddTag(sessionID string, tag string) (bool, error) {
	return s.app.Chat.AddTag(sessionID, tag)
}

func (s *ChatService) RemoveTag(sessionID string, tag string) (bool, error) {
	return s.app.Chat.RemoveTag(sessionID, tag)
}

func (s *ChatService) ListAllTags() ([][]any, error) {
	return s.app.Chat.ListAllTags()
}

func (s *ChatService) GetMessageSummary() (chat.MessageSummary, error) {
	return s.app.Chat.GetMessageSummary()
}

func (s *ChatService) LLMUsageGetTrends(days int, granularity string) ([]chat.UsageTrendPoint, error) {
	return s.app.Chat.LLMUsageGetTrends(days, granularity)
}

func (s *ChatService) LLMUsageByModel(startDate string, endDate string) ([]chat.ModelSummary, error) {
	return s.app.Chat.LLMUsageByModel(startDate, endDate)
}

func (s *ChatService) LLMUsageByCaller(startDate string, endDate string) ([]chat.CallerTypeSummary, error) {
	return s.app.Chat.LLMUsageByCaller(startDate, endDate)
}

func (s *ChatService) LLMUsageSummary(startDate *string, endDate *string) (chat.UsageSummary, error) {
	return s.app.Chat.LLMUsageSummary(startDate, endDate)
}

func (s *ChatService) LLMUsageRecent(limit int) ([]chat.UsageRecord, error) {
	return s.app.Chat.LLMUsageRecent(limit)
}

func (s *ChatService) LLMUsageDaily(startDate string, endDate string) ([]chat.DailySummary, error) {
	return s.app.Chat.LLMUsageDaily(startDate, endDate)
}

func (s *ChatService) LLMUsageCleanup(beforeDate string) (int, error) {
	return s.app.Chat.LLMUsageCleanup(beforeDate)
}

func (s *ChatService) BranchSession(sourceSessionID string, upToMessageID string) (chat.Session, error) {
	return s.app.Chat.BranchSession(sourceSessionID, upToMessageID)
}

func (s *ChatService) SendMessage(request chat.SendMessageRequest) (string, error) {
	return s.app.Chat.SendMessage(request)
}

func (s *ChatService) ContinueMessage(sessionID string, messageID string, options map[string]any) (string, error) {
	return s.app.Chat.ContinueMessage(sessionID, messageID, options)
}

func (s *ChatService) CancelStream(sessionID string, messageID string) (bool, error) {
	return s.app.Chat.CancelStream(sessionID, messageID)
}

func (s *ChatService) RetryMessage(sessionID string, messageID string, options map[string]any) (chat.RetryMessageResponse, error) {
	return s.app.Chat.RetryMessage(sessionID, messageID, options)
}

func (s *ChatService) EditAndResend(request chat.EditAndResendRequest) (chat.EditAndResendResponse, error) {
	return s.app.Chat.EditAndResend(request)
}

func (s *ChatService) RespondToolApproval(sessionID string, toolCallID string, toolName string, approved bool, reason *string, remember bool, arguments map[string]any) (bool, error) {
	return s.app.Chat.RespondToolApproval(sessionID, toolCallID, toolName, approved, reason, remember, arguments)
}

func (s *ChatService) ClearApprovalHistory() (int, error) {
	return s.app.Chat.ClearApprovalHistory()
}

func (s *ChatService) RespondAskUser(toolCallID string, selectedTexts []string, selectedIndices []int, customText *string, source string) (bool, error) {
	return s.app.Chat.RespondAskUser(toolCallID, selectedTexts, selectedIndices, customText, source)
}

func (s *ChatService) DeleteMessage(sessionID string, messageID string) (bool, error) {
	return s.app.Chat.DeleteMessage(sessionID, messageID)
}

func (s *ChatService) UpdateBlockContent(blockID string, content string) (bool, error) {
	return s.app.Chat.UpdateBlockContent(blockID, content)
}

func (s *ChatService) UpsertStreamingBlock(blockID string, messageID string, blockType string, content string, sessionID *string) (bool, error) {
	return s.app.Chat.UpsertStreamingBlock(blockID, messageID, blockType, content, sessionID)
}
