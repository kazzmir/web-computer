package coroutine

import (
	"errors"
	"testing"
	"time"
)

func waitForCoroutine(t *testing.T, c *Coroutine) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("coroutine did not exit after cancellation")
		return nil
	}
}

func TestStopBeforeRun(t *testing.T) {
	ran := false
	c := MakeCoroutine(func(YieldFunc) error { ran = true; return nil })
	c.Stop()
	c.Stop()
	if err := c.Run(); !errors.Is(err, CoroutineCancelled) {
		t.Fatalf("Run after Stop = %v", err)
	}
	if err := waitForCoroutine(t, c); !errors.Is(err, CoroutineCancelled) {
		t.Fatalf("Wait = %v", err)
	}
	if ran {
		t.Fatal("cancelled coroutine executed user code")
	}
	if err := c.Run(); !errors.Is(err, CoroutineCancelled) {
		t.Fatalf("Run = %v", err)
	}
}

func TestStopWhileYielded(t *testing.T) {
	cleaned := false
	c := MakeCoroutine(func(yield YieldFunc) error {
		defer func() { cleaned = true }()
		return yield()
	})
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	c.Stop()
	c.Stop()
	if err := waitForCoroutine(t, c); !errors.Is(err, CoroutineCancelled) {
		t.Fatalf("Wait = %v", err)
	}
	if !cleaned {
		t.Fatal("task cleanup did not run")
	}
}

func TestYieldAfterCancellationDoesNotBlock(t *testing.T) {
	c := MakeCoroutine(func(yield YieldFunc) error {
		// Even if a caller ignores its first cancellation error, another yield
		// must not block sending to an owner which has stopped running it.
		yield()
		return yield()
	})
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	c.Stop()
	if err := waitForCoroutine(t, c); !errors.Is(err, CoroutineCancelled) {
		t.Fatalf("Wait = %v", err)
	}
}

func TestStopConcurrentWithRun(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	c := MakeCoroutine(func(yield YieldFunc) error { close(started); <-release; return yield() })
	result := make(chan error, 1)
	go func() { result <- c.Run() }()
	<-started
	c.Stop()
	select {
	case err := <-result:
		if !errors.Is(err, CoroutineCancelled) {
			t.Errorf("Run = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Run did not respond to Stop")
	}
	close(release)
	if err := waitForCoroutine(t, c); !errors.Is(err, CoroutineCancelled) {
		t.Fatalf("Wait = %v", err)
	}
}

func TestCompletedResultIsStable(t *testing.T) {
	for _, result := range []error{nil, errors.New("task failed")} {
		c := MakeCoroutine(func(YieldFunc) error { return result })
		want := result
		if want == nil {
			want = CoroutineFinished
		}
		if err := c.Run(); err != want {
			t.Fatalf("Run = %v, want %v", err, want)
		}
		c.Stop()
		c.Stop()
		if err := waitForCoroutine(t, c); err != want {
			t.Fatalf("Wait = %v, want %v", err, want)
		}
		if err := c.Run(); err != want {
			t.Fatalf("repeated Run = %v, want %v", err, want)
		}
	}
}
