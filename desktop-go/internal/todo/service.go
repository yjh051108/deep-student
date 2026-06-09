package todo

import (
	"crypto/rand"
	"deep-student-go/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	statusPending   = "pending"
	statusCompleted = "completed"
	statusCancelled = "cancelled"

	priorityNone   = "none"
	priorityLow    = "low"
	priorityMedium = "medium"
	priorityHigh   = "high"
	priorityUrgent = "urgent"

	pomodoroTypeWork       = "work"
	pomodoroTypeShortBreak = "short_break"
	pomodoroTypeLongBreak  = "long_break"

	pomodoroStatusCompleted   = "completed"
	pomodoroStatusInterrupted = "interrupted"
)

var (
	validStatuses         = map[string]bool{statusPending: true, statusCompleted: true, statusCancelled: true}
	validPriorities       = map[string]bool{priorityNone: true, priorityLow: true, priorityMedium: true, priorityHigh: true, priorityUrgent: true}
	validPomodoroTypes    = map[string]bool{pomodoroTypeWork: true, pomodoroTypeShortBreak: true, pomodoroTypeLongBreak: true}
	validPomodoroStatuses = map[string]bool{pomodoroStatusCompleted: true, pomodoroStatusInterrupted: true}
	priorityRank          = map[string]int{priorityUrgent: 0, priorityHigh: 1, priorityMedium: 2, priorityLow: 3, priorityNone: 4}
)

type Service struct {
	mu    sync.RWMutex
	path  string
	state store
}

type store struct {
	Lists     []TodoList       `json:"lists"`
	Items     []TodoItem       `json:"items"`
	Pomodoros []PomodoroRecord `json:"pomodoros"`
}

type TodoList struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Color       *string `json:"color,omitempty"`
	SortOrder   int     `json:"sortOrder"`
	IsDefault   bool    `json:"isDefault"`
	IsFavorite  bool    `json:"isFavorite"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	DeletedAt   *string `json:"deletedAt,omitempty"`
}

type TodoItem struct {
	ID                 string  `json:"id"`
	TodoListID         string  `json:"todoListId"`
	Title              string  `json:"title"`
	Description        *string `json:"description,omitempty"`
	Status             string  `json:"status"`
	Priority           string  `json:"priority"`
	DueDate            *string `json:"dueDate,omitempty"`
	DueTime            *string `json:"dueTime,omitempty"`
	Reminder           *string `json:"reminder,omitempty"`
	TagsJSON           string  `json:"tagsJson"`
	SortOrder          int     `json:"sortOrder"`
	ParentID           *string `json:"parentId,omitempty"`
	CompletedAt        *string `json:"completedAt,omitempty"`
	RepeatJSON         *string `json:"repeatJson,omitempty"`
	AttachmentsJSON    string  `json:"attachmentsJson"`
	EstimatedPomodoros *int    `json:"estimatedPomodoros,omitempty"`
	CompletedPomodoros *int    `json:"completedPomodoros,omitempty"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
	DeletedAt          *string `json:"deletedAt,omitempty"`
}

type CreateTodoListInput struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type UpdateTodoListInput struct {
	ID          string  `json:"id"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type CreateTodoItemInput struct {
	TodoListID  string   `json:"todoListId"`
	Title       string   `json:"title"`
	Description *string  `json:"description,omitempty"`
	Priority    *string  `json:"priority,omitempty"`
	DueDate     *string  `json:"dueDate,omitempty"`
	DueTime     *string  `json:"dueTime,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	ParentID    *string  `json:"parentId,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type UpdateTodoItemInput struct {
	ID                 string   `json:"id"`
	Title              *string  `json:"title,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Status             *string  `json:"status,omitempty"`
	Priority           *string  `json:"priority,omitempty"`
	DueDate            *string  `json:"dueDate,omitempty"`
	DueTime            *string  `json:"dueTime,omitempty"`
	Reminder           *string  `json:"reminder,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	ParentID           *string  `json:"parentId,omitempty"`
	Attachments        []string `json:"attachments,omitempty"`
	RepeatJSON         *string  `json:"repeatJson,omitempty"`
	EstimatedPomodoros *int     `json:"estimatedPomodoros,omitempty"`
	CompletedPomodoros *int     `json:"completedPomodoros,omitempty"`
}

type ReorderItemsInput struct {
	ListID  string   `json:"listId"`
	ItemIDs []string `json:"itemIds"`
}

type TodoActiveSummary struct {
	TodayItems           []TodoSummaryItem `json:"todayItems"`
	OverdueItems         []TodoSummaryItem `json:"overdueItems"`
	UpcomingHighPriority []TodoSummaryItem `json:"upcomingHighPriority"`
	Stats                TodoStats         `json:"stats"`
}

type TodoSummaryItem struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Priority  string  `json:"priority"`
	DueDate   *string `json:"dueDate,omitempty"`
	DueTime   *string `json:"dueTime,omitempty"`
	ListTitle string  `json:"listTitle"`
}

type TodoStats struct {
	TotalPending   int `json:"totalPending"`
	TodayDue       int `json:"todayDue"`
	OverdueCount   int `json:"overdueCount"`
	TodayCompleted int `json:"todayCompleted"`
}

type PomodoroRecord struct {
	ID             string  `json:"id"`
	TodoItemID     *string `json:"todoItemId,omitempty"`
	StartTime      string  `json:"startTime"`
	EndTime        *string `json:"endTime,omitempty"`
	Duration       int     `json:"duration"`
	ActualDuration int     `json:"actualDuration"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
}

type CreatePomodoroInput struct {
	TodoItemID     *string `json:"todoItemId,omitempty"`
	StartTime      string  `json:"startTime"`
	EndTime        *string `json:"endTime,omitempty"`
	Duration       int     `json:"duration"`
	ActualDuration int     `json:"actualDuration"`
	Type           *string `json:"type,omitempty"`
	Status         *string `json:"status,omitempty"`
}

type PomodoroTodayStats struct {
	CompletedCount    int `json:"completedCount"`
	TotalFocusSeconds int `json:"totalFocusSeconds"`
	InterruptedCount  int `json:"interruptedCount"`
}

func NewService(dataDir string) (*Service, error) {
	service := &Service{
		path: filepath.Join(dataDir, "todo-go.json"),
		state: store{
			Lists:     []TodoList{},
			Items:     []TodoItem{},
			Pomodoros: []PomodoroRecord{},
		},
	}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) EnsureInbox() (TodoList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, list := range s.state.Lists {
		if list.DeletedAt == nil && list.IsDefault {
			return list, nil
		}
	}

	description := "默认待办列表"
	icon := "inbox"
	now := nowISO()
	list := TodoList{
		ID:          newID("tdl"),
		Title:       "收件箱",
		Description: &description,
		Icon:        &icon,
		SortOrder:   0,
		IsDefault:   true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.state.Lists = append(s.state.Lists, list)
	return list, s.flushLocked()
}

func (s *Service) CreateList(input CreateTodoListInput) (TodoList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "收件箱"
	}
	now := nowISO()
	list := TodoList{
		ID:          newID("tdl"),
		Title:       title,
		Description: input.Description,
		Icon:        input.Icon,
		Color:       input.Color,
		SortOrder:   nextListSortOrder(s.state.Lists),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.state.Lists = append(s.state.Lists, list)
	return list, s.flushLocked()
}

func (s *Service) GetList(listID string) (*TodoList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx := s.findListIndex(listID); idx >= 0 {
		list := s.state.Lists[idx]
		return &list, nil
	}
	return nil, nil
}

func (s *Service) ListLists() []TodoList {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lists := make([]TodoList, 0, len(s.state.Lists))
	for _, list := range s.state.Lists {
		if list.DeletedAt == nil {
			lists = append(lists, list)
		}
	}
	sort.SliceStable(lists, func(i, j int) bool {
		if lists[i].IsDefault != lists[j].IsDefault {
			return lists[i].IsDefault
		}
		if lists[i].SortOrder != lists[j].SortOrder {
			return lists[i].SortOrder < lists[j].SortOrder
		}
		return lists[i].UpdatedAt > lists[j].UpdatedAt
	})
	return lists
}

func (s *Service) UpdateList(input UpdateTodoListInput) (TodoList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findListIndex(input.ID)
	if idx < 0 {
		return TodoList{}, notFound("TodoList", input.ID)
	}

	list := s.state.Lists[idx]
	if input.Title != nil {
		list.Title = *input.Title
	}
	if input.Description != nil {
		list.Description = input.Description
	}
	if input.Icon != nil {
		list.Icon = input.Icon
	}
	if input.Color != nil {
		list.Color = input.Color
	}
	list.UpdatedAt = nowISO()
	s.state.Lists[idx] = list
	return list, s.flushLocked()
}

func (s *Service) DeleteList(listID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findListIndex(listID)
	if idx < 0 {
		if s.hasListRecord(listID) {
			return nil
		}
		return notFound("TodoList", listID)
	}
	if s.state.Lists[idx].IsDefault {
		return errors.New("Cannot delete the default inbox list")
	}

	now := nowISO()
	s.state.Lists[idx].DeletedAt = &now
	s.state.Lists[idx].UpdatedAt = now
	for i := range s.state.Items {
		if s.state.Items[i].TodoListID == listID && s.state.Items[i].DeletedAt == nil {
			s.state.Items[i].DeletedAt = &now
			s.state.Items[i].UpdatedAt = now
		}
	}
	return s.flushLocked()
}

func (s *Service) ToggleListFavorite(listID string) (TodoList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findListIndex(listID)
	if idx < 0 {
		return TodoList{}, notFound("TodoList", listID)
	}
	list := s.state.Lists[idx]
	list.IsFavorite = !list.IsFavorite
	list.UpdatedAt = nowISO()
	s.state.Lists[idx] = list
	return list, s.flushLocked()
}

func (s *Service) CreateItem(input CreateTodoItemInput) (TodoItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return TodoItem{}, errors.New("Todo item title cannot be empty")
	}
	if s.findListIndex(input.TodoListID) < 0 {
		return TodoItem{}, notFound("TodoList", input.TodoListID)
	}

	priority := priorityNone
	if input.Priority != nil {
		priority = *input.Priority
	}
	if !validPriorities[priority] {
		return TodoItem{}, invalidValue("priority", priority, validPriorityNames())
	}

	parentID := normalizePtr(input.ParentID)
	if parentID != nil {
		parent, err := s.requireItem(*parentID)
		if err != nil {
			return TodoItem{}, err
		}
		if parent.TodoListID != input.TodoListID {
			return TodoItem{}, fmt.Errorf("Parent item belongs to list '%s', expected '%s'", parent.TodoListID, input.TodoListID)
		}
	}

	now := nowISO()
	item := TodoItem{
		ID:              newID("ti"),
		TodoListID:      input.TodoListID,
		Title:           title,
		Description:     input.Description,
		Status:          statusPending,
		Priority:        priority,
		DueDate:         normalizePtr(input.DueDate),
		DueTime:         normalizePtr(input.DueTime),
		TagsJSON:        stringArrayJSON(input.Tags),
		SortOrder:       nextItemSortOrder(s.state.Items, input.TodoListID, parentID),
		ParentID:        parentID,
		AttachmentsJSON: stringArrayJSON(input.Attachments),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.state.Items = append(s.state.Items, item)
	s.touchListLocked(input.TodoListID, now)
	return item, s.flushLocked()
}

func (s *Service) GetItem(itemID string) (*TodoItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx := s.findItemIndex(itemID); idx >= 0 {
		item := s.state.Items[idx]
		return &item, nil
	}
	return nil, nil
}

func (s *Service) ListItems(listID string, includeCompleted bool) []TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]TodoItem, 0)
	for _, item := range s.state.Items {
		if item.DeletedAt != nil || item.TodoListID != listID {
			continue
		}
		if !includeCompleted && item.Status == statusCompleted {
			continue
		}
		items = append(items, item)
	}
	sortTodoItems(items)
	return items
}

func (s *Service) UpdateItem(input UpdateTodoItemInput) (TodoItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findItemIndex(input.ID)
	if idx < 0 {
		return TodoItem{}, notFound("TodoItem", input.ID)
	}

	current := s.state.Items[idx]
	item := current
	now := nowISO()

	if input.Title != nil {
		item.Title = *input.Title
	}
	if strings.TrimSpace(item.Title) == "" {
		return TodoItem{}, errors.New("Todo item title cannot be empty")
	}
	if input.Description != nil {
		item.Description = input.Description
	}
	if input.Status != nil {
		item.Status = *input.Status
	}
	if !validStatuses[item.Status] {
		return TodoItem{}, invalidValue("status", item.Status, validStatusNames())
	}
	if input.Priority != nil {
		item.Priority = *input.Priority
	}
	if !validPriorities[item.Priority] {
		return TodoItem{}, invalidValue("priority", item.Priority, validPriorityNames())
	}
	if input.DueDate != nil {
		item.DueDate = normalizePtr(input.DueDate)
	}
	if input.DueTime != nil {
		item.DueTime = normalizePtr(input.DueTime)
	}
	if input.Reminder != nil {
		item.Reminder = normalizePtr(input.Reminder)
	}
	if input.Tags != nil {
		item.TagsJSON = stringArrayJSON(input.Tags)
	}
	if input.ParentID != nil {
		parentID := normalizePtr(input.ParentID)
		if parentID != nil {
			if *parentID == input.ID {
				return TodoItem{}, errors.New("Cannot set parent_id to self")
			}
			parent, err := s.requireItem(*parentID)
			if err != nil {
				return TodoItem{}, err
			}
			if parent.TodoListID != item.TodoListID {
				return TodoItem{}, fmt.Errorf("Parent item belongs to list '%s', expected '%s'", parent.TodoListID, item.TodoListID)
			}
			if s.isDescendantLocked(input.ID, *parentID) {
				return TodoItem{}, errors.New("Cannot set parent_id to a descendant item")
			}
		}
		item.ParentID = parentID
	}
	if input.Attachments != nil {
		item.AttachmentsJSON = stringArrayJSON(input.Attachments)
	}
	if input.RepeatJSON != nil {
		item.RepeatJSON = input.RepeatJSON
	}
	if input.EstimatedPomodoros != nil {
		item.EstimatedPomodoros = input.EstimatedPomodoros
	}
	if input.CompletedPomodoros != nil {
		item.CompletedPomodoros = input.CompletedPomodoros
	}

	if item.Status == statusCompleted && current.Status != statusCompleted {
		item.CompletedAt = &now
	} else if item.Status != statusCompleted {
		item.CompletedAt = nil
	}

	item.UpdatedAt = now
	s.state.Items[idx] = item
	s.touchListLocked(item.TodoListID, now)
	return item, s.flushLocked()
}

func (s *Service) ToggleItem(itemID string) (TodoItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findItemIndex(itemID)
	if idx < 0 {
		return TodoItem{}, notFound("TodoItem", itemID)
	}

	now := nowISO()
	item := s.state.Items[idx]
	if item.Status == statusCompleted {
		item.Status = statusPending
		item.CompletedAt = nil
	} else {
		item.Status = statusCompleted
		item.CompletedAt = &now
	}
	item.UpdatedAt = now
	s.state.Items[idx] = item
	s.touchListLocked(item.TodoListID, now)
	return item, s.flushLocked()
}

func (s *Service) DeleteItem(itemID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findItemIndex(itemID)
	if idx < 0 {
		if s.hasItemRecord(itemID) {
			return nil
		}
		return notFound("TodoItem", itemID)
	}

	now := nowISO()
	s.state.Items[idx].DeletedAt = &now
	s.state.Items[idx].UpdatedAt = now
	s.touchListLocked(s.state.Items[idx].TodoListID, now)
	return s.flushLocked()
}

func (s *Service) ReorderItems(input ReorderItemsInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.findListIndex(input.ListID) < 0 {
		return notFound("TodoList", input.ListID)
	}
	now := nowISO()
	for order, id := range input.ItemIDs {
		for i := range s.state.Items {
			if s.state.Items[i].DeletedAt == nil && s.state.Items[i].ID == id && s.state.Items[i].TodoListID == input.ListID {
				s.state.Items[i].SortOrder = order
				s.state.Items[i].UpdatedAt = now
				break
			}
		}
	}
	s.touchListLocked(input.ListID, now)
	return s.flushLocked()
}

func (s *Service) ListToday(includeCompleted bool) []TodoItem {
	today := todayDate()
	return s.filterItems(func(item TodoItem) bool {
		if item.DueDate == nil || *item.DueDate != today {
			return false
		}
		return item.Status == statusPending || (includeCompleted && item.Status == statusCompleted)
	}, sortDueToday)
}

func (s *Service) ListOverdue(includeCompleted bool) []TodoItem {
	today := todayDate()
	return s.filterItems(func(item TodoItem) bool {
		if item.DueDate == nil || *item.DueDate >= today {
			return false
		}
		return item.Status == statusPending || (includeCompleted && item.Status == statusCompleted)
	}, sortDueDateAsc)
}

func (s *Service) ListUpcoming(days int, includeCompleted bool) []TodoItem {
	today := todayDate()
	endDate := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	return s.filterItems(func(item TodoItem) bool {
		if item.DueDate == nil || *item.DueDate <= today || *item.DueDate > endDate {
			return false
		}
		return item.Status == statusPending || includeCompleted && item.Status == statusCompleted
	}, sortDueDateAsc)
}

func (s *Service) ListCompleted(listID *string) []TodoItem {
	return s.filterItems(func(item TodoItem) bool {
		if item.Status != statusCompleted {
			return false
		}
		return listID == nil || item.TodoListID == *listID
	}, sortCompletedDesc)
}

func (s *Service) Search(query string) []TodoItem {
	needle := strings.ToLower(query)
	return limitItems(s.filterItems(func(item TodoItem) bool {
		if needle == "" {
			return true
		}
		return strings.Contains(strings.ToLower(item.Title), needle) ||
			(item.Description != nil && strings.Contains(strings.ToLower(*item.Description), needle))
	}, sortUpdatedDesc), 50)
}

func (s *Service) ActiveSummary() (*TodoActiveSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	listTitles := make(map[string]string)
	for _, list := range s.state.Lists {
		if list.DeletedAt == nil {
			listTitles[list.ID] = list.Title
		}
	}
	if len(listTitles) == 0 {
		return nil, nil
	}

	today := todayDate()
	upcomingEnd := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	var pending, todayDue, overdue, todayCompleted int
	var todayItems, overdueItems, upcoming []TodoSummaryItem

	for _, item := range s.state.Items {
		if item.DeletedAt != nil {
			continue
		}
		listTitle, listActive := listTitles[item.TodoListID]
		if !listActive {
			continue
		}
		if item.Status == statusPending {
			pending++
			if item.DueDate != nil && *item.DueDate == today {
				todayDue++
				todayItems = append(todayItems, summaryItem(item, listTitle))
			}
			if item.DueDate != nil && *item.DueDate < today {
				overdue++
				overdueItems = append(overdueItems, summaryItem(item, listTitle))
			}
			if item.DueDate != nil && *item.DueDate > today && *item.DueDate <= upcomingEnd && (item.Priority == priorityUrgent || item.Priority == priorityHigh) {
				upcoming = append(upcoming, summaryItem(item, listTitle))
			}
		}
		if item.Status == statusCompleted && item.CompletedAt != nil && sameLocalDate(*item.CompletedAt, today) {
			todayCompleted++
		}
	}

	if pending == 0 && todayCompleted == 0 {
		return nil, nil
	}

	sort.SliceStable(todayItems, func(i, j int) bool {
		return prioritySortKey(todayItems[i].Priority) < prioritySortKey(todayItems[j].Priority)
	})
	sort.SliceStable(overdueItems, func(i, j int) bool {
		if dateValue(overdueItems[i].DueDate) != dateValue(overdueItems[j].DueDate) {
			return dateValue(overdueItems[i].DueDate) > dateValue(overdueItems[j].DueDate)
		}
		return overdueItems[i].Title < overdueItems[j].Title
	})
	sort.SliceStable(upcoming, func(i, j int) bool {
		if dateValue(upcoming[i].DueDate) != dateValue(upcoming[j].DueDate) {
			return dateValue(upcoming[i].DueDate) < dateValue(upcoming[j].DueDate)
		}
		return prioritySortKey(upcoming[i].Priority) < prioritySortKey(upcoming[j].Priority)
	})

	return &TodoActiveSummary{
		TodayItems:           limitSummary(todayItems, 5),
		OverdueItems:         limitSummary(overdueItems, 3),
		UpcomingHighPriority: limitSummary(upcoming, 3),
		Stats: TodoStats{
			TotalPending:   pending,
			TodayDue:       todayDue,
			OverdueCount:   overdue,
			TodayCompleted: todayCompleted,
		},
	}, nil
}

func (s *Service) CreatePomodoroRecord(input CreatePomodoroInput) (PomodoroRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordType := pomodoroTypeWork
	if input.Type != nil {
		recordType = *input.Type
	}
	if !validPomodoroTypes[recordType] {
		return PomodoroRecord{}, invalidValue("pomodoro type", recordType, validPomodoroTypeNames())
	}

	recordStatus := pomodoroStatusCompleted
	if input.Status != nil {
		recordStatus = *input.Status
	}
	if !validPomodoroStatuses[recordStatus] {
		return PomodoroRecord{}, invalidValue("pomodoro status", recordStatus, validPomodoroStatusNames())
	}

	if strings.TrimSpace(input.StartTime) == "" {
		return PomodoroRecord{}, errors.New("pomodoro startTime cannot be empty")
	}
	if input.Duration < 0 || input.ActualDuration < 0 {
		return PomodoroRecord{}, errors.New("pomodoro duration cannot be negative")
	}

	todoItemID := normalizePtr(input.TodoItemID)
	if todoItemID != nil && s.findItemIndex(*todoItemID) < 0 {
		return PomodoroRecord{}, notFound("TodoItem", *todoItemID)
	}

	now := nowISO()
	record := PomodoroRecord{
		ID:             newID("pd"),
		TodoItemID:     todoItemID,
		StartTime:      strings.TrimSpace(input.StartTime),
		EndTime:        normalizePtr(input.EndTime),
		Duration:       input.Duration,
		ActualDuration: input.ActualDuration,
		Type:           recordType,
		Status:         recordStatus,
		CreatedAt:      now,
	}
	s.state.Pomodoros = append(s.state.Pomodoros, record)

	if todoItemID != nil && record.Type == pomodoroTypeWork && record.Status == pomodoroStatusCompleted {
		idx := s.findItemIndex(*todoItemID)
		if idx >= 0 {
			if s.state.Items[idx].CompletedPomodoros == nil {
				zero := 0
				s.state.Items[idx].CompletedPomodoros = &zero
			}
			*s.state.Items[idx].CompletedPomodoros += 1
			s.state.Items[idx].UpdatedAt = now
			s.touchListLocked(s.state.Items[idx].TodoListID, now)
		}
	}

	return record, s.flushLocked()
}

func (s *Service) GetPomodoroRecord(recordID string) (*PomodoroRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, record := range s.state.Pomodoros {
		if record.ID == recordID {
			out := record
			return &out, nil
		}
	}
	return nil, nil
}

func (s *Service) ListPomodorosByTodo(todoItemID string) []PomodoroRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]PomodoroRecord, 0)
	for _, record := range s.state.Pomodoros {
		if record.TodoItemID != nil && *record.TodoItemID == todoItemID {
			records = append(records, record)
		}
	}
	sortPomodoroCreatedDesc(records)
	return records
}

func (s *Service) PomodoroTodayStats() PomodoroTodayStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := todayDate()
	var stats PomodoroTodayStats
	for _, record := range s.state.Pomodoros {
		if !sameLocalDate(record.CreatedAt, today) || record.Type != pomodoroTypeWork {
			continue
		}
		if record.Status == pomodoroStatusCompleted {
			stats.CompletedCount++
			stats.TotalFocusSeconds += record.ActualDuration
		} else if record.Status == pomodoroStatusInterrupted {
			stats.InterruptedCount++
		}
	}
	return stats
}

func (s *Service) ListTodayPomodoros() []PomodoroRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := todayDate()
	records := make([]PomodoroRecord, 0)
	for _, record := range s.state.Pomodoros {
		if sameLocalDate(record.CreatedAt, today) {
			records = append(records, record)
		}
	}
	sortPomodoroCreatedDesc(records)
	return records
}

func (s *Service) filterItems(predicate func(TodoItem) bool, sorter func([]TodoItem)) []TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]TodoItem, 0)
	for _, item := range s.state.Items {
		if item.DeletedAt == nil && predicate(item) {
			items = append(items, item)
		}
	}
	sorter(items)
	return items
}

func (s *Service) findListIndex(listID string) int {
	for i, list := range s.state.Lists {
		if list.ID == listID && list.DeletedAt == nil {
			return i
		}
	}
	return -1
}

func (s *Service) hasListRecord(listID string) bool {
	for _, list := range s.state.Lists {
		if list.ID == listID {
			return true
		}
	}
	return false
}

func (s *Service) findItemIndex(itemID string) int {
	for i, item := range s.state.Items {
		if item.ID == itemID && item.DeletedAt == nil {
			return i
		}
	}
	return -1
}

func (s *Service) hasItemRecord(itemID string) bool {
	for _, item := range s.state.Items {
		if item.ID == itemID {
			return true
		}
	}
	return false
}

func (s *Service) requireItem(itemID string) (TodoItem, error) {
	idx := s.findItemIndex(itemID)
	if idx < 0 {
		return TodoItem{}, notFound("TodoItem (parent)", itemID)
	}
	return s.state.Items[idx], nil
}

func (s *Service) touchListLocked(listID, timestamp string) {
	for i := range s.state.Lists {
		if s.state.Lists[i].ID == listID && s.state.Lists[i].DeletedAt == nil {
			s.state.Lists[i].UpdatedAt = timestamp
			return
		}
	}
}

func (s *Service) isDescendantLocked(itemID, candidateParentID string) bool {
	children := make(map[string][]string)
	for _, item := range s.state.Items {
		if item.DeletedAt == nil && item.ParentID != nil {
			children[*item.ParentID] = append(children[*item.ParentID], item.ID)
		}
	}

	stack := append([]string{}, children[itemID]...)
	for len(stack) > 0 {
		next := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if next == candidateParentID {
			return true
		}
		stack = append(stack, children[next]...)
	}
	return false
}

func (s *Service) load() error {
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(bytes, &s.state); err != nil {
		return err
	}
	if s.state.Lists == nil {
		s.state.Lists = []TodoList{}
	}
	if s.state.Items == nil {
		s.state.Items = []TodoItem{}
	}
	if s.state.Pomodoros == nil {
		s.state.Pomodoros = []PomodoroRecord{}
	}
	return nil
}

func (s *Service) flushLocked() error {
	return storage.WriteJSONAtomic(s.path, s.state)
}

func nextListSortOrder(lists []TodoList) int {
	maxOrder := -1
	for _, list := range lists {
		if list.DeletedAt == nil && list.SortOrder > maxOrder {
			maxOrder = list.SortOrder
		}
	}
	return maxOrder + 1
}

func nextItemSortOrder(items []TodoItem, listID string, parentID *string) int {
	maxOrder := -1
	for _, item := range items {
		if item.DeletedAt != nil || item.TodoListID != listID {
			continue
		}
		if ptrEqual(item.ParentID, parentID) && item.SortOrder > maxOrder {
			maxOrder = item.SortOrder
		}
	}
	return maxOrder + 1
}

func ptrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func normalizePtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func stringArrayJSON(values []string) string {
	if values == nil {
		values = []string{}
	}
	bytes, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

func nowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func todayDate() string {
	return time.Now().Format("2006-01-02")
}

func sameLocalDate(timestamp string, date string) bool {
	trimmed := strings.TrimSpace(timestamp)
	if trimmed == "" {
		return false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.Local().Format("2006-01-02") == date
		}
	}
	return strings.HasPrefix(trimmed, date)
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, randomToken(10))
}

func randomToken(length int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_-"
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = alphabet[int(time.Now().UnixNano())%len(alphabet)]
			continue
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out)
}

func invalidValue(param, value string, expected []string) error {
	return fmt.Errorf("Unsupported todo %s '%s'; expected one of %v", param, value, expected)
}

func validStatusNames() []string {
	return []string{statusPending, statusCompleted, statusCancelled}
}

func validPriorityNames() []string {
	return []string{priorityNone, priorityLow, priorityMedium, priorityHigh, priorityUrgent}
}

func validPomodoroTypeNames() []string {
	return []string{pomodoroTypeWork, pomodoroTypeShortBreak, pomodoroTypeLongBreak}
}

func validPomodoroStatusNames() []string {
	return []string{pomodoroStatusCompleted, pomodoroStatusInterrupted}
}

func notFound(resource, id string) error {
	return fmt.Errorf("%s not found: %s", resource, id)
}

func prioritySortKey(priority string) int {
	if rank, ok := priorityRank[priority]; ok {
		return rank
	}
	return priorityRank[priorityNone]
}

func sortTodoItems(items []TodoItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SortOrder != items[j].SortOrder {
			return items[i].SortOrder < items[j].SortOrder
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
}

func sortDueToday(items []TodoItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return items[i].Status == statusPending
		}
		if prioritySortKey(items[i].Priority) != prioritySortKey(items[j].Priority) {
			return prioritySortKey(items[i].Priority) < prioritySortKey(items[j].Priority)
		}
		if timeValue(items[i].DueTime) != timeValue(items[j].DueTime) {
			return timeValue(items[i].DueTime) < timeValue(items[j].DueTime)
		}
		return items[i].SortOrder < items[j].SortOrder
	})
}

func sortDueDateAsc(items []TodoItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if dateValue(items[i].DueDate) != dateValue(items[j].DueDate) {
			return dateValue(items[i].DueDate) < dateValue(items[j].DueDate)
		}
		if items[i].Status != items[j].Status {
			return items[i].Status == statusPending
		}
		if prioritySortKey(items[i].Priority) != prioritySortKey(items[j].Priority) {
			return prioritySortKey(items[i].Priority) < prioritySortKey(items[j].Priority)
		}
		return items[i].SortOrder < items[j].SortOrder
	})
}

func sortCompletedDesc(items []TodoItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if textValue(items[i].CompletedAt) != textValue(items[j].CompletedAt) {
			return textValue(items[i].CompletedAt) > textValue(items[j].CompletedAt)
		}
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
}

func sortUpdatedDesc(items []TodoItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
}

func sortPomodoroCreatedDesc(records []PomodoroRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})
}

func dateValue(value *string) string {
	if value == nil {
		return "9999-12-31"
	}
	return *value
}

func timeValue(value *string) string {
	if value == nil {
		return "99:99"
	}
	return *value
}

func textValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func limitItems(items []TodoItem, max int) []TodoItem {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func limitSummary(items []TodoSummaryItem, max int) []TodoSummaryItem {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func summaryItem(item TodoItem, listTitle string) TodoSummaryItem {
	return TodoSummaryItem{
		ID:        item.ID,
		Title:     item.Title,
		Priority:  item.Priority,
		DueDate:   item.DueDate,
		DueTime:   item.DueTime,
		ListTitle: listTitle,
	}
}
