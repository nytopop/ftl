package ftl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var Debug = false

// StateHolder has full control over state loading and
// unloading.
type StateHolder interface {
	StateLoader
	StateUnloader
}

// StateLoader can load state.
type StateLoader interface {
	// Load a unit of state.
	//
	// Intended for state units that will be unloaded
	// by the caller.
	Load() (loaded bool, unload func())

	// Load an unload tasklet.
	//
	// Intended for state units that are meant to stay
	// loaded until the state unloader begins unloading.
	LoadUnload(unload Tasklet) (loaded bool)
}

// StateUnloader can unload state, and allow/disallow
// further loads.
type StateUnloader interface {
	// Accepts sets whether the state loader will accept
	// state loads.
	//
	// The state unloader must not begin unloading if
	// it is currently accepting new loads.
	Accepts(accept bool)

	// Unload any local state units.
	Unload(ctx context.Context) error

	// UnloadWait concurrently calls Unload and Wait.
	//
	// Essentially, it performs a full unload of all
	// state.
	UnloadWait(ctx context.Context) error

	// Wait for remote state units to be unloaded.
	Wait(ctx context.Context) error
}

var _ StateHolder = new(State)

// State keeps track of some abstract 'units of state'.
//
// It's like a togglable sync.WaitGroup that also keeps
// track of a parbound remote unload function.
type State struct {
	unloads Tasklet // this is inefficient in the extreme. maybe just remove?
	states  uint64
	accept  bool
	mu      sync.Mutex
}

func (s *State) unloadSingle() {
	s.mu.Lock()
	s.states--
	if Debug {
		fmt.Println("--", s.states)
	}
	s.mu.Unlock()
}

// Load a single unit of state.
//
// If loaded is true, unload must be called to de-allocate
// the loaded state. The unload func is idempotent, calling
// it more than once does nothing. Internally, we expect that
// once unload has been called, the caller has gotten rid of
// or dealt with whatever state they were holding.
//
// If loaded is false, unload will be nil. The caller must
// not acquire any state in this situation - it indicates that
// Accept(false) has been called.
func (s *State) Load() (loaded bool, unload func()) {
	s.mu.Lock()

	// if we're accepting, it's safe to add some state
	if s.accept {
		s.states++
		if Debug {
			fmt.Println("++", s.states)
		}

		// set the unload
		loaded = true
		var once sync.Once
		unload = func() { once.Do(s.unloadSingle) }
	}

	s.mu.Unlock()
	return loaded, unload
}

// LoadUnload loads a caller provided unloading tasklet, queuing
// it to be executed on graceful shutdown.
func (s *State) LoadUnload(unload Tasklet) (loaded bool) {
	s.mu.Lock()

	// if we're accepting, it's safe to add some state
	if s.accept {
		s.states++
		if Debug {
			fmt.Println("++fn", s.states)
		}

		unload = Tasklet.Seq(
			func(_ context.Context) error {
				s.unloadSingle()
				return nil
			},
			unload,
		).Once()

		switch s.unloads {
		case nil:
			// if there are no unloads loaded, just do unload
			s.unloads = unload
		default:
			// if there are, parallelize
			s.unloads = Tasklet.Par(
				s.unloads,
				unload,
			)
		}

		loaded = true
	}

	s.mu.Unlock()
	return loaded
}

// Accepts changes whether the State will accept any further
// state loads. It always succeeds.
//
// Call Accepts(false) if you want to pause state loading - after
// this, loads will return loaded=false.
//
// Call Accepts(true) if you want to resume state loading - after
// this, loads will return loaded=true.
func (s *State) Accepts(accept bool) {
	s.mu.Lock()
	s.accept = accept
	s.mu.Unlock()
}

// Unload any loaded unload tasklets.
func (s *State) Unload(ctx context.Context) error {
	if s.unloads != nil {
		return s.unloads(ctx)
	}
	return nil
}

func (s *State) UnloadWait(ctx context.Context) error {
	return Tasklet.Par(s.Unload, s.Wait)(ctx)
}

// Wait until all remote state has been unloaded. It
// respects cancellation of the passed in context.
//
// If the context is cancelled, Wait returns the underlying
// context error. If it finishes waiting before the context
// was cancelled, it returns nil. If the context is never
// cancelled, it will block until it succeeds.
//
// Calling Wait while still accepting state loads violates
// the underlying invariants, and should never, ever happen.
func (s *State) Wait(ctx context.Context) error {
	var (
		// errDone never escapes this tasklet
		errDone = errors.New("")

		// succeedIfLoaded returns nil if still loaded, or
		// errDone if everything is unloaded.
		succeedIfLoaded Tasklet = func(_ context.Context) error {
			if s.accept {
				panic("wait called while accepting state loads")
			}
			if s.states == 0 {
				return errDone
			}
			return nil
		}

		delayer Tasklet = func(_ context.Context) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		}

		//      |- tasklet gives us automatic context checking
		//      v
		loop Tasklet = Tasklet.Seq(
			succeedIfLoaded.Mu(&s.mu), // check if loaded
			delayer,                   // don't burn cpu
		).While(Nil()) // repeat until something happens
	)

	if err := loop(ctx); err == errDone {
		return nil
	} else {
		return err
	}
}
