package qbank

import (
	"strconv"
	"sync"
	"testing"
)

// TestSubmitConcurrentRace 校验并发 Submit 时 mastery 计数不出现负值/越界。
// BUG-001 回归：在 -race 下必须稳定通过。
func TestSubmitConcurrentRace(t *testing.T) {
	s := newSvc(t)
	for i := 0; i < 50; i++ {
		set := &Set{ID: idstr("set", i), Questions: []Question{}}
		for j := 0; j < 10; j++ {
			set.Questions = append(set.Questions, Question{
				ID: idstr("q", j), Answer: "a", Knowledge: []string{idstr("k", i*10+j)},
			})
		}
		s.mu.Lock()
		s.sets[set.ID] = set
		s.mu.Unlock()
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			setID := idstr("set", i)
			a, _ := s.StartAttempt(setID)
			for j := 0; j < 10; j++ {
				s.Answer(a.ID, idstr("q", j), "a")
			}
			s.Submit(a.ID)
		}(i)
	}
	wg.Wait()
	m := s.Mastery()
	if len(m) != 500 {
		t.Fatalf("mastery count=%d, expected 500", len(m))
	}
	for k, v := range m {
		if v < 0 || v > 100 {
			t.Fatalf("mastery[%s]=%d out of range", k, v)
		}
	}
}

// TestSubmitMasteryConcurrent 同时跑 Submit 与 Mastery（背景线程持续读），
// 进一步覆盖 RLock/RLock 与 Submit 内部写锁的冲突。-race 下应无数据竞争。
func TestSubmitMasteryConcurrent(t *testing.T) {
	s := newSvc(t)
	for i := 0; i < 20; i++ {
		set := &Set{ID: idstr("set", i), Questions: []Question{}}
		for j := 0; j < 5; j++ {
			set.Questions = append(set.Questions, Question{
				ID: idstr("q", j), Answer: "a", Knowledge: []string{idstr("k", i*5+j)},
			})
		}
		s.mu.Lock()
		s.sets[set.ID] = set
		s.mu.Unlock()
	}
	stop := make(chan struct{})
	var bg sync.WaitGroup
	bg.Add(1)
	go func() {
		defer bg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = s.Mastery()
		}
	}()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			setID := idstr("set", i)
			a, _ := s.StartAttempt(setID)
			for j := 0; j < 5; j++ {
				s.Answer(a.ID, idstr("q", j), "a")
			}
			s.Submit(a.ID)
		}(i)
	}
	wg.Wait()
	close(stop)
	bg.Wait()
}

func idstr(p string, i int) string {
	return p + "_" + strconv.Itoa(i)
}
