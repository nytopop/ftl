package fsync

import (
	"context"
	"runtime"
	"sort"
	"sync"

	"github.com/nytopop/ftl"
)

var _ sync.Locker = new(nop)

type nop struct{ int }

func (n *nop) Lock()   {}
func (n *nop) Unlock() {}

// Executor is a priority queue based concurrent
// worker pool.

// Executor is a bounded worker pool implementation.
type Executor struct {
	// n workers
	n uint32

	// queued jobs from calls to Add,
	// lazily sorted
	queued   *safeQueue
	flush    []Task
	inflight chan Task

	// sync / runtime
	wg      *sync.WaitGroup
	fcond   *sync.Cond
	state   ftl.StateLoader
	stateMu *sync.RWMutex
}

const (
	CPUBound  uint16 = 1
	IOBound   uint16 = 64
	IdleBound uint16 = 4096
)

func NewExecutor(goScale uint16) *Executor {
	var n uint32 = 1
	if goScale != 0 {
		n = uint32(runtime.NumCPU() * int(goScale))
	}

	return &Executor{
		n: n,

		queued:   newQueue(int(8 * n)),
		flush:    nil,
		inflight: nil,

		wg:      new(sync.WaitGroup),
		fcond:   sync.NewCond(new(nop)),
		state:   nil,
		stateMu: new(sync.RWMutex),
	}
}

func (e *Executor) taskWorker(ctx context.Context) func() {
	return func() {
		for task := range e.inflight {
			if err := task.Action(ctx); err != nil {
				if task.Retry != nil && task.Retry(err) {
					e.queued.push(task)
					return
				}
			}
			task.unload()
		}
	}
}

// push transfers tasks from the queued pq to the inflight
// channel. the reason it is necessary is to actually execute
// tasks in priority order - however, once a task has made it
// into the inflight channel its completion can happen at any
// time. using a small buffer in the inflight channel allows
// for low priority tasks to occasionally be executed before
// high priority tasks if contention on resources is low.
func (e *Executor) flushWorker() {
	n := e.queued.flush(e.flush)
	if n > 0 {
		for _, t := range e.flush[:n] {
			e.inflight <- t
		}
	} else {
		e.fcond.Wait()
	}
}

// spawn spawns a goroutine that will repeatedly
// execute f until the shared context expires.
func (e *Executor) spawn(ctx context.Context, f func()) {
	e.wg.Add(1)

	go func() {
	start:
		select {
		case <-ctx.Done():
			e.wg.Done()
			return
		default:
			f()
		}
		goto start
	}()
}

func (e *Executor) Add(t Task) (loaded bool) {
	e.stateMu.RLock()
	loaded, unload := e.state.Load()
	e.stateMu.RUnlock()
	if !loaded {
		return false
	}

	t.unload = unload
	e.queued.push(t)
	e.fcond.Signal()

	return true
}

func (e *Executor) Routine(ctx context.Context, state ftl.StateLoader) error {
	e.stateMu.Lock()
	e.state = state
	eMut := &Executor{
		n: e.n,

		// maintain the parent queue - calls to Add
		// must use the same queue
		queued: e.queued,

		// items within one flush buffer are sorted for
		// priority scheduling
		flush: make([]Task, 8*e.n),

		// if the working set is < e.n ,
		// priority doesn't matter - just buffer it
		inflight: make(chan Task, e.n),

		wg:    e.wg,
		fcond: e.fcond,
		state: e.state,
	}
	e.stateMu.Unlock()

	// spawn everything
	for i := 0; i < int(e.n); i++ {
		eMut.spawn(ctx,
			eMut.taskWorker(ctx),
		)
	}
	eMut.spawn(ctx, eMut.flushWorker)

	// block until cancelled
	<-ctx.Done()

	eMut.fcond.Broadcast() // kill push
	close(eMut.inflight)   // kill pop

	// wait for exit of all goroutines
	eMut.wg.Wait()

	return nil
}

type Task struct {
	// Action to execute. Required.
	Action ftl.Tasklet

	// Priority of the task; larger values
	// will be scheduled first.
	Priority int

	// Retry determines whether to re-schedule
	// the action.
	Retry ftl.Predicate

	// unload will be executed after success.
	unload func()
}

type safeQueue struct {
	ts taskQueue
	mu sync.Mutex
}

func newQueue(preAlloc int) *safeQueue {
	return &safeQueue{
		ts: make(taskQueue, 0, preAlloc),
	}
}

// push a single element onto the queue. doesn't maintain
// sort order, this call is meant to block for the shortest
// possible amount of time.
func (s *safeQueue) push(t Task) {
	s.mu.Lock()
	s.ts = append(s.ts, t)
	s.mu.Unlock()
}

// flush into the provided buffer. the number of
// flushed elements will be returned.
func (s *safeQueue) flush(buf []Task) (n int) {
	s.mu.Lock()
	sort.Sort(s.ts)
	switch {
	case len(s.ts) >= len(buf):
		n = len(buf)
		copy(buf, s.ts[:n])
		s.ts = s.ts[n:]
	case len(s.ts) > 0:
		n = len(s.ts)
		copy(buf, s.ts)
		s.ts = nil
	}
	s.mu.Unlock()

	return n
}

var _ sort.Interface = (taskQueue)(nil)

type taskQueue []Task

func (ts taskQueue) Len() int {
	return len(ts)
}

func (ts taskQueue) Less(i, j int) bool {
	return ts[i].Priority > ts[j].Priority
}

func (ts taskQueue) Swap(i, j int) {
	ts[i], ts[j] = ts[j], ts[i]
}
