package fsrs

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/helixnow/deep-student-go/pkg/store"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st)
}

func TestAddAndDue(t *testing.T) {
	s := newSvc(t)
	cards, err := s.AddCards("英语", []CardInput{
		{Front: "apple", Back: "苹果"},
		{Front: "banana", Back: "香蕉"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 2 {
		t.Fatalf("cards=%d", len(cards))
	}
	due, err := s.DueCards("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 2 {
		t.Fatalf("due=%d", len(due))
	}
	cnt, _ := s.DueCount()
	if cnt != 2 {
		t.Fatalf("dueCount=%d", cnt)
	}
}

func TestReviewSchedule(t *testing.T) {
	s := newSvc(t)
	cards, _ := s.AddCards("物理", []CardInput{{Front: "E=mc²", Back: "质能方程"}})
	cid := cards[0].CardID

	// Good → 进入复习，间隔拉长
	st, err := s.Review(cid, Good)
	if err != nil {
		t.Fatal(err)
	}
	if st.State != "learning" {
		t.Fatalf("state=%s", st.State)
	}
	if st.Stability <= 0 || st.Reps != 1 {
		t.Fatalf("st=%+v", st)
	}
	// 应已不在到期列表
	due, _ := s.DueCards("", 10)
	for _, d := range due {
		if d.CardID == cid {
			t.Fatal("card should not be due after Good")
		}
	}
	// Again → 10 分钟内重新到期（分钟级间隔）
	st, _ = s.Review(cid, Again)
	soon := time.Now().UTC().Add(15 * time.Minute)
	if !st.DueAt.Before(soon) {
		t.Fatalf("dueAt=%v (should be within 15min)", st.DueAt)
	}
	// 复查 lapses
	st2, _ := s.Get(cid)
	if st2.Lapses != 1 {
		t.Fatalf("lapses=%d", st2.Lapses)
	}
}

func TestReviewLogs(t *testing.T) {
	s := newSvc(t)
	cards, _ := s.AddCards("数学", []CardInput{{Front: "1+1", Back: "2"}})
	cid := cards[0].CardID
	s.Review(cid, Good)
	s.Review(cid, Hard)
	logs, err := s.ReviewLogs(cid, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("logs=%d", len(logs))
	}
	if logs[0].Rating != Hard {
		t.Fatalf("latest=%+v", logs[0])
	}
}

func TestDeckStatsAndDelete(t *testing.T) {
	s := newSvc(t)
	cards, _ := s.AddCards("A", []CardInput{{Front: "1", Back: "2"}, {Front: "3", Back: "4"}})
	s.AddCards("B", []CardInput{{Front: "5", Back: "6"}})
	stats, err := s.DeckStats()
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 {
		t.Fatalf("stats=%+v", stats)
	}
	// 删除一张
	if err := s.Delete(cards[0].CardID); err != nil {
		t.Fatal(err)
	}
	all, _ := s.AllCards("", 100)
	if len(all) != 2 {
		t.Fatalf("all=%d", len(all))
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	st, _ := store.Open(filepath.Join(dir, "x.db"))
	s := New(st)
	cards, _ := s.AddCards("P", []CardInput{{Front: "f", Back: "b"}})
	s.Review(cards[0].CardID, Good)
	st.Close()

	st2, _ := store.Open(filepath.Join(dir, "x.db"))
	defer st2.Close()
	s2 := New(st2)
	all, err := s2.AllCards("", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Reps != 1 {
		t.Fatalf("persisted=%+v", all)
	}
}

