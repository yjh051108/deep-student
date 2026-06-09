package bindings

import (
	"deep-student-go/internal/app"
	"deep-student-go/internal/todo"
)

type TodoService struct {
	app *app.App
}

func NewTodoService(app *app.App) *TodoService {
	return &TodoService{app: app}
}

func (s *TodoService) EnsureInbox() (todo.TodoList, error) {
	return s.app.Todo.EnsureInbox()
}

func (s *TodoService) CreateList(input todo.CreateTodoListInput) (todo.TodoList, error) {
	return s.app.Todo.CreateList(input)
}

func (s *TodoService) GetList(listID string) (*todo.TodoList, error) {
	return s.app.Todo.GetList(listID)
}

func (s *TodoService) ListLists() ([]todo.TodoList, error) {
	return s.app.Todo.ListLists(), nil
}

func (s *TodoService) UpdateList(input todo.UpdateTodoListInput) (todo.TodoList, error) {
	return s.app.Todo.UpdateList(input)
}

func (s *TodoService) DeleteList(listID string) error {
	return s.app.Todo.DeleteList(listID)
}

func (s *TodoService) ToggleListFavorite(listID string) (todo.TodoList, error) {
	return s.app.Todo.ToggleListFavorite(listID)
}

func (s *TodoService) CreateItem(input todo.CreateTodoItemInput) (todo.TodoItem, error) {
	return s.app.Todo.CreateItem(input)
}

func (s *TodoService) GetItem(itemID string) (*todo.TodoItem, error) {
	return s.app.Todo.GetItem(itemID)
}

func (s *TodoService) ListItems(listID string, includeCompleted bool) ([]todo.TodoItem, error) {
	return s.app.Todo.ListItems(listID, includeCompleted), nil
}

func (s *TodoService) UpdateItem(input todo.UpdateTodoItemInput) (todo.TodoItem, error) {
	return s.app.Todo.UpdateItem(input)
}

func (s *TodoService) ToggleItem(itemID string) (todo.TodoItem, error) {
	return s.app.Todo.ToggleItem(itemID)
}

func (s *TodoService) DeleteItem(itemID string) error {
	return s.app.Todo.DeleteItem(itemID)
}

func (s *TodoService) ReorderItems(input todo.ReorderItemsInput) error {
	return s.app.Todo.ReorderItems(input)
}

func (s *TodoService) ListToday(includeCompleted bool) ([]todo.TodoItem, error) {
	return s.app.Todo.ListToday(includeCompleted), nil
}

func (s *TodoService) ListOverdue(includeCompleted bool) ([]todo.TodoItem, error) {
	return s.app.Todo.ListOverdue(includeCompleted), nil
}

func (s *TodoService) ListUpcoming(days int, includeCompleted bool) ([]todo.TodoItem, error) {
	return s.app.Todo.ListUpcoming(days, includeCompleted), nil
}

func (s *TodoService) ListCompleted(listID *string) ([]todo.TodoItem, error) {
	return s.app.Todo.ListCompleted(listID), nil
}

func (s *TodoService) Search(query string) ([]todo.TodoItem, error) {
	return s.app.Todo.Search(query), nil
}

func (s *TodoService) ActiveSummary() (*todo.TodoActiveSummary, error) {
	return s.app.Todo.ActiveSummary()
}

func (s *TodoService) CreatePomodoroRecord(input todo.CreatePomodoroInput) (todo.PomodoroRecord, error) {
	return s.app.Todo.CreatePomodoroRecord(input)
}

func (s *TodoService) GetPomodoroRecord(recordID string) (*todo.PomodoroRecord, error) {
	return s.app.Todo.GetPomodoroRecord(recordID)
}

func (s *TodoService) ListPomodorosByTodo(todoItemID string) ([]todo.PomodoroRecord, error) {
	return s.app.Todo.ListPomodorosByTodo(todoItemID), nil
}

func (s *TodoService) PomodoroTodayStats() (todo.PomodoroTodayStats, error) {
	return s.app.Todo.PomodoroTodayStats(), nil
}

func (s *TodoService) ListTodayPomodoros() ([]todo.PomodoroRecord, error) {
	return s.app.Todo.ListTodayPomodoros(), nil
}
