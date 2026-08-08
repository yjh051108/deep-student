// todo 包的业务层：列表/条目 CRUD、回收站、视图查询、AI 拆解。
//
// 构造函数 New(vfs, store, llmReg) 与既有 internal 包保持一致；
// 持久化完全交给 Store（SQLite），Service 只做参数校验与 LLM 辅助。

package todo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/helixnow/deep-student-go/pkg/llm"
	"github.com/helixnow/deep-student-go/pkg/store"
	"github.com/helixnow/deep-student-go/pkg/vfs"
)

// Service Todo 业务服务。
type Service struct {
	store  *Store
	llmReg *llm.Registry
	mu     sync.RWMutex
}

// New 构造 Todo 服务。vfs 参数保留用于与既有构造签名一致（暂未使用）。
// 自动执行表迁移；迁移失败仅打印到 stderr，不阻塞构造。
func New(_ *vfs.FS, s *store.Store, llmReg *llm.Registry) *Service {
	if llmReg == nil {
		llmReg = llm.NewRegistry()
	}
	ts := NewStore(s)
	if err := ts.Migrate(); err != nil {
		fmt.Printf("[todo] migrate failed: %v\n", err)
	}
	return &Service{store: ts, llmReg: llmReg}
}

// ErrNotFound 通用未找到。
var ErrNotFound = errors.New("todo: not found")

// EnsureInbox 确保内置收件箱存在（幂等），返回其 ID。
func (s *Service) EnsureInbox() (*List, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lists, err := s.store.ListLists(true)
	if err != nil {
		return nil, err
	}
	for _, l := range lists {
		if l.IsInbox {
			return l, nil
		}
	}
	now := time.Now().UTC()
	inbox := &List{
		ID:        uuid.NewString(),
		Name:      "收件箱",
		IsInbox:   true,
		SortOrder: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.CreateList(inbox); err != nil {
		return nil, err
	}
	return inbox, nil
}

// ===================== 列表 =====================

// CreateList 创建列表。
func (s *Service) CreateList(p CreateListParams) (*List, error) {
	if strings.TrimSpace(p.Name) == "" {
		return nil, errors.New("todo: list name required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	l := &List{
		ID:        uuid.NewString(),
		Name:      strings.TrimSpace(p.Name),
		Color:     p.Color,
		Icon:      p.Icon,
		SortOrder: 100,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.CreateList(l); err != nil {
		return nil, err
	}
	return l, nil
}

// GetList 读取列表（含统计）。
func (s *Service) GetList(id string) (*List, error) {
	l, err := s.store.GetList(id)
	if err != nil {
		return nil, err
	}
	s.fillListStats(l)
	return l, nil
}

// ListLists 列出列表（含统计）。
func (s *Service) ListLists(includeDeleted bool) ([]List, error) {
	lists, err := s.store.ListLists(includeDeleted)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(lists))
	for _, l := range lists {
		ids = append(ids, l.ID)
	}
	counts, err := s.store.ListCounts(ids)
	if err != nil {
		return nil, err
	}
	out := make([]List, 0, len(lists))
	for _, l := range lists {
		if c, ok := counts[l.ID]; ok {
			l.ItemCount = c[0]
			l.PendingCount = c[1]
			l.CompletedCount = c[2]
		}
		out = append(out, *l)
	}
	return out, nil
}

// UpdateList 更新列表。
func (s *Service) UpdateList(p UpdateListParams) (*List, error) {
	if p.Name != nil && strings.TrimSpace(*p.Name) == "" {
		return nil, errors.New("todo: list name cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.UpdateList(p); err != nil {
		return nil, err
	}
	return s.store.GetList(p.ID)
}

// DeleteList 软删除列表（回收站）。
func (s *Service) DeleteList(id string) error {
	return s.store.SoftDeleteList(id)
}

// RestoreList 恢复列表。
func (s *Service) RestoreList(id string) error {
	return s.store.RestoreList(id)
}

// PurgeList 彻底删除列表。
func (s *Service) PurgeList(id string) error {
	return s.store.PurgeList(id)
}

// PurgeDeletedLists 清空回收站列表，返回删除数量。
func (s *Service) PurgeDeletedLists() (int64, error) {
	return s.store.PurgeDeletedLists()
}

// ListDeletedLists 列出回收站中的列表。
func (s *Service) ListDeletedLists() ([]List, error) {
	lists, err := s.store.ListLists(true)
	if err != nil {
		return nil, err
	}
	out := make([]List, 0, len(lists))
	for _, l := range lists {
		if l.IsDeleted {
			out = append(out, *l)
		}
	}
	return out, nil
}

// ===================== 条目 =====================

// CreateItem 创建条目。
func (s *Service) CreateItem(p CreateItemParams) (*Item, error) {
	if strings.TrimSpace(p.Title) == "" {
		return nil, errors.New("todo: item title required")
	}
	if p.ListID == "" {
		inbox, err := s.EnsureInbox()
		if err != nil {
			return nil, err
		}
		p.ListID = inbox.ID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// 校验父条目属于同一列表（防跨列表挂子任务）
	if p.ParentID != nil {
		parent, err := s.store.GetItem(*p.ParentID)
		if err != nil {
			return nil, fmt.Errorf("todo: parent not found: %w", err)
		}
		if parent.ListID != p.ListID {
			return nil, errors.New("todo: parent item in different list")
		}
	}
	now := time.Now().UTC()
	it := &Item{
		ID:            uuid.NewString(),
		ListID:        p.ListID,
		Title:         strings.TrimSpace(p.Title),
		Notes:         p.Notes,
		DueAt:         p.DueAt,
		Priority:      p.Priority,
		Tags:          p.Tags,
		ParentID:      p.ParentID,
		EstPomodoros:  p.EstPomodoros,
		Repeat:        p.Repeat,
		RemindAt:      p.RemindAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if it.Tags == nil {
		it.Tags = []string{}
	}
	if err := s.store.CreateItem(it); err != nil {
		return nil, err
	}
	return it, nil
}

// GetItem 读取条目。
func (s *Service) GetItem(id string) (*Item, error) {
	it, err := s.store.GetItem(id)
	if err != nil {
		return nil, err
	}
	s.fillItemStats(it)
	return it, nil
}

// ListItems 列出条目（可选列表与状态过滤）。
func (s *Service) ListItems(listID string, filter ItemFilter) ([]Item, error) {
	items, err := s.store.ListItems(listID, filter)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	subs, err := s.store.SubCounts(ids)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		it.SubCount = subs[it.ID]
		out = append(out, *it)
	}
	return out, nil
}

// UpdateItem 更新条目。
func (s *Service) UpdateItem(p UpdateItemParams) (*Item, error) {
	if p.Title != nil && strings.TrimSpace(*p.Title) == "" {
		return nil, errors.New("todo: item title cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.UpdateItem(p); err != nil {
		return nil, err
	}
	return s.store.GetItem(p.ID)
}

// ToggleItem 切换完成状态，返回切换后的条目。
func (s *Service) ToggleItem(id string) (*Item, error) {
	it, err := s.store.GetItem(id)
	if err != nil {
		return nil, err
	}
	var at *time.Time
	if it.CompletedAt == nil {
		t := time.Now().UTC()
		at = &t
	}
	if err := s.store.SetCompleted(id, at); err != nil {
		return nil, err
	}
	return s.store.GetItem(id)
}

// DeleteItem 软删除条目（回收站）。
func (s *Service) DeleteItem(id string) error {
	return s.store.SoftDeleteItem(id)
}

// RestoreItem 恢复条目。
func (s *Service) RestoreItem(id string) error {
	return s.store.RestoreItem(id)
}

// PurgeItem 彻底删除条目。
func (s *Service) PurgeItem(id string) error {
	return s.store.PurgeItem(id)
}

// PurgeDeletedItems 清空回收站条目。
func (s *Service) PurgeDeletedItems() (int64, error) {
	return s.store.PurgeDeletedItems()
}

// ListDeletedItems 列出回收站条目。
func (s *Service) ListDeletedItems() ([]Item, error) {
	items, err := s.store.ListItems("", FilterDeleted)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		out = append(out, *it)
	}
	return out, nil
}

// ReorderItems 重排条目顺序。
func (s *Service) ReorderItems(listID string, ids []string) error {
	return s.store.ReorderItems(listID, ids)
}

// ===================== 视图查询 =====================

// ListToday 今日到期待办。
func (s *Service) ListToday() ([]Item, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)
	return s.itemsBetween(&start, &end)
}

// ListOverdue 逾期未办（截止早于今天开始时刻）。
func (s *Service) ListOverdue() ([]Item, error) {
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := startOfToday.Add(-time.Nanosecond)
	return s.itemsBetween(nil, &end)
}

// ListUpcoming 未来 7 天待办。
func (s *Service) ListUpcoming() ([]Item, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7).Add(-time.Nanosecond)
	return s.itemsBetween(&start, &end)
}

// ListReminders 最近到提醒时间的条目。
func (s *Service) ListReminders(limit int) ([]Item, error) {
	items, err := s.store.ListItems("", FilterPending)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]Item, 0)
	for _, it := range items {
		if it.RemindAt != nil && !it.RemindAt.After(now) {
			out = append(out, *it)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// Search 搜索待办。
func (s *Service) Search(keyword string, limit int) ([]Item, error) {
	items, err := s.store.Search(strings.TrimSpace(keyword), limit)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		out = append(out, *it)
	}
	return out, nil
}

// Summary 活跃待办总览。
func (s *Service) Summary() (*Summary, error) {
	pending, err := s.store.CountPending()
	if err != nil {
		return nil, err
	}
	completed, err := s.store.CountCompleted()
	if err != nil {
		return nil, err
	}
	overdue, err := s.ListOverdue()
	if err != nil {
		return nil, err
	}
	today, err := s.ListToday()
	if err != nil {
		return nil, err
	}
	lists, err := s.ListLists(false)
	if err != nil {
		return nil, err
	}
	return &Summary{
		TotalPending:   pending,
		TotalCompleted: completed,
		OverdueCount:   len(overdue),
		DueTodayCount:  len(today),
		Lists:          lists,
	}, nil
}

// ===================== AI 拆解 =====================

// AIBreakdown 用 LLM 把一个复杂任务拆成子任务列表（含预估番茄数）。
// 返回的子任务作为 Item（未入库），由调用方决定是否批量创建。
func (s *Service) AIBreakdown(ctx context.Context, title, notes string) ([]Item, error) {
	provider, ok := s.llmReg.Get("openai")
	if !ok {
		return nil, errors.New("todo: llm provider unavailable")
	}
	prompt := `你是任务拆解助手。请把下面的任务拆解为 3-6 个可执行的子任务，只输出 JSON 数组，不要任何其它文字：
[
  {"title":"子任务标题","estPomodoros":1}
]
任务：` + title
	if strings.TrimSpace(notes) != "" {
		prompt += "\n补充说明：" + notes
	}
	resp, err := provider.Chat(ctx, llm.ChatRequest{
		Model:       "gpt-4o-mini",
		Messages:    []llm.Message{{Role: llm.RoleUser, Content: prompt}},
		Temperature: 0.3,
	})
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Title        string `json:"title"`
		EstPomodoros int    `json:"estPomodoros"`
	}
	content := strings.TrimSpace(resp.Content)
	start, end := strings.Index(content, "["), strings.LastIndex(content, "]")
	if start < 0 || end < start {
		return nil, errors.New("todo: llm did not return a JSON array")
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &raw); err != nil {
		return nil, fmt.Errorf("todo: parse breakdown: %w", err)
	}
	out := make([]Item, 0, len(raw))
	now := time.Now().UTC()
	for _, r := range raw {
		if strings.TrimSpace(r.Title) == "" {
			continue
		}
		out = append(out, Item{
			ID:            uuid.NewString(),
			Title:         strings.TrimSpace(r.Title),
			EstPomodoros:  r.EstPomodoros,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	if len(out) == 0 {
		return nil, errors.New("todo: breakdown produced no subtasks")
	}
	return out, nil
}

// ===================== 内部辅助 =====================

func (s *Service) itemsBetween(start, end *time.Time) ([]Item, error) {
	items, err := s.store.ListDueBetween(start, end)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		out = append(out, *it)
	}
	return out, nil
}

func (s *Service) fillListStats(l *List) {
	counts, err := s.store.ListCounts([]string{l.ID})
	if err != nil {
		return
	}
	if c, ok := counts[l.ID]; ok {
		l.ItemCount = c[0]
		l.PendingCount = c[1]
		l.CompletedCount = c[2]
	}
}

func (s *Service) fillItemStats(it *Item) {
	subs, err := s.store.SubCounts([]string{it.ID})
	if err != nil {
		return
	}
	it.SubCount = subs[it.ID]
}
