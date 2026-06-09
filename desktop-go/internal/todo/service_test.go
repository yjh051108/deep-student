package todo

import (
	"strings"
	"testing"
	"time"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func strPtr(value string) *string {
	return &value
}

func TestEnsureInboxIsIdempotent(t *testing.T) {
	service := newTestService(t)

	first, err := service.EnsureInbox()
	if err != nil {
		t.Fatalf("EnsureInbox() error = %v", err)
	}
	second, err := service.EnsureInbox()
	if err != nil {
		t.Fatalf("EnsureInbox() second error = %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("EnsureInbox() created duplicate inbox: %s != %s", first.ID, second.ID)
	}
	if !first.IsDefault || first.Title != "收件箱" {
		t.Fatalf("unexpected inbox: %+v", first)
	}
}

func TestCreateItemRejectsCrossListParent(t *testing.T) {
	service := newTestService(t)
	listA, err := service.CreateList(CreateTodoListInput{Title: "A"})
	if err != nil {
		t.Fatalf("CreateList(A) error = %v", err)
	}
	listB, err := service.CreateList(CreateTodoListInput{Title: "B"})
	if err != nil {
		t.Fatalf("CreateList(B) error = %v", err)
	}
	parent, err := service.CreateItem(CreateTodoItemInput{TodoListID: listA.ID, Title: "parent"})
	if err != nil {
		t.Fatalf("CreateItem(parent) error = %v", err)
	}

	_, err = service.CreateItem(CreateTodoItemInput{
		TodoListID: listB.ID,
		Title:      "child",
		ParentID:   &parent.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "Parent item belongs to list") {
		t.Fatalf("expected cross-list parent error, got %v", err)
	}
}

func TestListTodayHonorsIncludeCompleted(t *testing.T) {
	service := newTestService(t)
	list, err := service.CreateList(CreateTodoListInput{Title: "Today"})
	if err != nil {
		t.Fatalf("CreateList() error = %v", err)
	}
	today := time.Now().Format("2006-01-02")

	pending, err := service.CreateItem(CreateTodoItemInput{
		TodoListID: list.ID,
		Title:      "pending",
		DueDate:    &today,
	})
	if err != nil {
		t.Fatalf("CreateItem(pending) error = %v", err)
	}
	completed, err := service.CreateItem(CreateTodoItemInput{
		TodoListID: list.ID,
		Title:      "completed",
		DueDate:    &today,
	})
	if err != nil {
		t.Fatalf("CreateItem(completed) error = %v", err)
	}
	if _, err := service.ToggleItem(completed.ID); err != nil {
		t.Fatalf("ToggleItem() error = %v", err)
	}

	pendingOnly := service.ListToday(false)
	if len(pendingOnly) != 1 || pendingOnly[0].ID != pending.ID {
		t.Fatalf("ListToday(false) = %+v, want only %s", pendingOnly, pending.ID)
	}

	withCompleted := service.ListToday(true)
	if len(withCompleted) != 2 {
		t.Fatalf("ListToday(true) len = %d, want 2: %+v", len(withCompleted), withCompleted)
	}
}

func TestUpdateItemRejectsParentCycle(t *testing.T) {
	service := newTestService(t)
	list, err := service.CreateList(CreateTodoListInput{Title: "Cycle"})
	if err != nil {
		t.Fatalf("CreateList() error = %v", err)
	}
	parent, err := service.CreateItem(CreateTodoItemInput{TodoListID: list.ID, Title: "parent"})
	if err != nil {
		t.Fatalf("CreateItem(parent) error = %v", err)
	}
	child, err := service.CreateItem(CreateTodoItemInput{
		TodoListID: list.ID,
		Title:      "child",
		ParentID:   &parent.ID,
	})
	if err != nil {
		t.Fatalf("CreateItem(child) error = %v", err)
	}

	_, err = service.UpdateItem(UpdateTodoItemInput{ID: parent.ID, ParentID: &child.ID})
	if err == nil || !strings.Contains(err.Error(), "descendant") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestActiveSummaryReturnsPendingAndCompletedStats(t *testing.T) {
	service := newTestService(t)
	list, err := service.EnsureInbox()
	if err != nil {
		t.Fatalf("EnsureInbox() error = %v", err)
	}
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	high := priorityHigh

	if _, err := service.CreateItem(CreateTodoItemInput{TodoListID: list.ID, Title: "today", DueDate: &today, Priority: &high}); err != nil {
		t.Fatalf("CreateItem(today) error = %v", err)
	}
	if _, err := service.CreateItem(CreateTodoItemInput{TodoListID: list.ID, Title: "late", DueDate: &yesterday}); err != nil {
		t.Fatalf("CreateItem(late) error = %v", err)
	}
	done, err := service.CreateItem(CreateTodoItemInput{TodoListID: list.ID, Title: "done", DueDate: &today})
	if err != nil {
		t.Fatalf("CreateItem(done) error = %v", err)
	}
	if _, err := service.ToggleItem(done.ID); err != nil {
		t.Fatalf("ToggleItem(done) error = %v", err)
	}

	summary, err := service.ActiveSummary()
	if err != nil {
		t.Fatalf("ActiveSummary() error = %v", err)
	}
	if summary == nil {
		t.Fatal("ActiveSummary() = nil, want stats")
	}
	if summary.Stats.TotalPending != 2 || summary.Stats.TodayDue != 1 || summary.Stats.OverdueCount != 1 || summary.Stats.TodayCompleted != 1 {
		t.Fatalf("unexpected stats: %+v", summary.Stats)
	}
	if len(summary.TodayItems) != 1 || summary.TodayItems[0].ListTitle != list.Title {
		t.Fatalf("unexpected today items: %+v", summary.TodayItems)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	service, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	list, err := service.CreateList(CreateTodoListInput{Title: "Persist", Description: strPtr("kept")})
	if err != nil {
		t.Fatalf("CreateList() error = %v", err)
	}
	if _, err := service.CreateItem(CreateTodoItemInput{TodoListID: list.ID, Title: "task", Tags: []string{"a"}}); err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}

	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService(reloaded) error = %v", err)
	}
	lists := reloaded.ListLists()
	if len(lists) != 1 || lists[0].ID != list.ID || lists[0].Description == nil || *lists[0].Description != "kept" {
		t.Fatalf("unexpected reloaded lists: %+v", lists)
	}
	items := reloaded.ListItems(list.ID, true)
	if len(items) != 1 || items[0].Title != "task" || items[0].TagsJSON != `["a"]` {
		t.Fatalf("unexpected reloaded items: %+v", items)
	}
}

func TestCreatePomodoroRecordUpdatesTodoAndStats(t *testing.T) {
	service := newTestService(t)
	list, err := service.CreateList(CreateTodoListInput{Title: "Focus"})
	if err != nil {
		t.Fatalf("CreateList() error = %v", err)
	}
	item, err := service.CreateItem(CreateTodoItemInput{TodoListID: list.ID, Title: "read"})
	if err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}

	record, err := service.CreatePomodoroRecord(CreatePomodoroInput{
		TodoItemID:     &item.ID,
		StartTime:      time.Now().Add(-25 * time.Minute).Format(time.RFC3339),
		EndTime:        strPtr(time.Now().Format(time.RFC3339)),
		Duration:       1500,
		ActualDuration: 1500,
	})
	if err != nil {
		t.Fatalf("CreatePomodoroRecord() error = %v", err)
	}
	if record.ID == "" || record.Type != pomodoroTypeWork || record.Status != pomodoroStatusCompleted {
		t.Fatalf("unexpected record defaults: %+v", record)
	}

	reloadedItem, err := service.GetItem(item.ID)
	if err != nil {
		t.Fatalf("GetItem() error = %v", err)
	}
	if reloadedItem == nil || reloadedItem.CompletedPomodoros == nil || *reloadedItem.CompletedPomodoros != 1 {
		t.Fatalf("completed pomodoros not incremented: %+v", reloadedItem)
	}

	byTodo := service.ListPomodorosByTodo(item.ID)
	if len(byTodo) != 1 || byTodo[0].ID != record.ID {
		t.Fatalf("ListPomodorosByTodo() = %+v, want record %s", byTodo, record.ID)
	}

	stats := service.PomodoroTodayStats()
	if stats.CompletedCount != 1 || stats.TotalFocusSeconds != 1500 || stats.InterruptedCount != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	today := service.ListTodayPomodoros()
	if len(today) != 1 || today[0].ID != record.ID {
		t.Fatalf("ListTodayPomodoros() = %+v, want record %s", today, record.ID)
	}
}

func TestInterruptedPomodoroDoesNotIncrementTodo(t *testing.T) {
	service := newTestService(t)
	list, err := service.CreateList(CreateTodoListInput{Title: "Focus"})
	if err != nil {
		t.Fatalf("CreateList() error = %v", err)
	}
	item, err := service.CreateItem(CreateTodoItemInput{TodoListID: list.ID, Title: "read"})
	if err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}
	status := pomodoroStatusInterrupted

	if _, err := service.CreatePomodoroRecord(CreatePomodoroInput{
		TodoItemID:     &item.ID,
		StartTime:      time.Now().Format(time.RFC3339),
		Duration:       1500,
		ActualDuration: 300,
		Status:         &status,
	}); err != nil {
		t.Fatalf("CreatePomodoroRecord() error = %v", err)
	}

	reloadedItem, err := service.GetItem(item.ID)
	if err != nil {
		t.Fatalf("GetItem() error = %v", err)
	}
	if reloadedItem == nil || reloadedItem.CompletedPomodoros != nil {
		t.Fatalf("interrupted pomodoro should not increment todo: %+v", reloadedItem)
	}

	stats := service.PomodoroTodayStats()
	if stats.CompletedCount != 0 || stats.TotalFocusSeconds != 0 || stats.InterruptedCount != 1 {
		t.Fatalf("unexpected interrupted stats: %+v", stats)
	}
}
