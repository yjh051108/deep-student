package bindings

import (
	"deep-student-go/internal/anki"
	"deep-student-go/internal/app"
	"testing"
)

func TestAnkiServiceSaveAnkiCardsDelegatesToAppService(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEEP_STUDENT_DATA_DIR", dir)

	application, err := app.New()
	if err != nil {
		t.Fatal(err)
	}
	service := NewAnkiService(application)
	documentID := "doc-binding"
	front := "Binding front"
	back := "Binding back"

	response, err := service.SaveAnkiCards(anki.SaveAnkiCardsRequest{
		DocumentID: &documentID,
		Cards: []anki.SaveAnkiCardPayload{{
			Front: &front,
			Back:  &back,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.TaskID == "" || len(response.SavedIDs) != 1 || response.SavedIDs[0] == "" {
		t.Fatalf("unexpected response: %+v", response)
	}

	cards, err := application.Anki.GetDocumentCards(documentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || cards[0]["front"] != front || cards[0]["back"] != back {
		t.Fatalf("binding did not persist card through app service: %+v", cards)
	}
}
