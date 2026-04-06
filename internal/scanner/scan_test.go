package scanner

import (
	"sync"
	"testing"
)

func TestTryStart_FirstCall(t *testing.T) {
	state := &ScanState{}

	if !state.TryStart() {
		t.Error("expected TryStart to return true on first call")
	}

	if !state.IsRunning() {
		t.Error("expected IsRunning to be true after TryStart")
	}
}

func TestTryStart_WhileRunning(t *testing.T) {
	state := &ScanState{}
	state.TryStart()

	if state.TryStart() {
		t.Error("expected TryStart to return false while already running")
	}
}

func TestTryStart_Concurrent(t *testing.T) {
	state := &ScanState{}

	var wg sync.WaitGroup
	starts := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if state.TryStart() {
				mu.Lock()
				starts++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if starts != 1 {
		t.Errorf("expected exactly 1 successful TryStart, got %d", starts)
	}
}

func TestStatus(t *testing.T) {
	state := &ScanState{}
	state.TryStart()

	state.mu.Lock()
	state.Total = 100
	state.Processed = 42
	state.Errors = 3
	state.mu.Unlock()

	status := state.Status()

	if !status.Running {
		t.Error("expected Running to be true")
	}
	if status.Total != 100 {
		t.Errorf("expected Total 100, got %d", status.Total)
	}
	if status.Processed != 42 {
		t.Errorf("expected Processed 42, got %d", status.Processed)
	}
	if status.Errors != 3 {
		t.Errorf("expected Errors 3, got %d", status.Errors)
	}
}
