package coroutine

import (
	"errors"
	"sync"
)

// Coroutine runs a cooperative task one yield at a time. Run calls are
// serialized; Stop may be called concurrently and is safe to call repeatedly.
// The task must propagate errors returned by its YieldFunc to finish promptly.
type Coroutine struct {
	resume    chan struct{}
	yielded   chan struct{}
	cancelled chan struct{}
	done      chan struct{}
	stopOnce  sync.Once
	runMu     sync.Mutex
	err       error // Written by the task; read only after done is closed.
}

type YieldFunc func() error

type AcceptYieldFunc func(yield YieldFunc) error

var CoroutineFinished = errors.New("coroutine finished")
var CoroutineCancelled = errors.New("coroutine cancelled")

func MakeCoroutine(user AcceptYieldFunc) *Coroutine {
	coro := &Coroutine{
		resume:    make(chan struct{}),
		yielded:   make(chan struct{}),
		cancelled: make(chan struct{}),
		done:      make(chan struct{}),
		err:       CoroutineFinished,
	}
	go func() {
		defer close(coro.done)
		// Stopping a task before its first Run must not execute user code.
		select {
		case <-coro.cancelled:
			coro.err = CoroutineCancelled
			return
		case <-coro.resume:
		}
		err := user(func() error {
			// Never send another yield to an owner which has already shut down.
			select {
			case <-coro.cancelled:
				return CoroutineCancelled
			default:
			}
			select {
			case <-coro.cancelled:
				return CoroutineCancelled
			case coro.yielded <- struct{}{}:
			}
			select {
			case <-coro.cancelled:
				return CoroutineCancelled
			case <-coro.resume:
				return nil
			}
		})
		if err != nil {
			coro.err = err
		}
		select {
		case <-coro.cancelled:
			coro.err = CoroutineCancelled
		default:
		}
	}()
	return coro
}

// Stop requests cancellation and wakes a task suspended at yield. Call Wait
// before closing resources used by the task. Cancellation is cooperative.
func (coro *Coroutine) Stop() {
	coro.stopOnce.Do(func() { close(coro.cancelled) })
}

// Wait joins the task and returns its terminal result.
func (coro *Coroutine) Wait() error {
	<-coro.done
	return coro.err
}

// Run resumes the task until its next yield or completion. A nil result means
// it yielded; CoroutineFinished means it returned nil. Cancellation returns
// CoroutineCancelled. After completion subsequent calls return the same result.
func (coro *Coroutine) Run() error {
	coro.runMu.Lock()
	defer coro.runMu.Unlock()
	select {
	case <-coro.done:
		return coro.err
	default:
	}
	select {
	case <-coro.cancelled:
		return CoroutineCancelled
	default:
	}
	select {
	case <-coro.done:
		return coro.err
	case <-coro.cancelled:
		return CoroutineCancelled
	case coro.resume <- struct{}{}:
	}
	select {
	case <-coro.done:
		return coro.err
	case <-coro.cancelled:
		return CoroutineCancelled
	case <-coro.yielded:
		return nil
	}
}
