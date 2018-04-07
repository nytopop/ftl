package ftl

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// Routine represents a potentially long-lived, stateful, interruptible
// computation.
//
// It's a magical contraption that can be composed with other such
// magical contraptions. When executed, the final magical contraption
// is still just as state-safe and interruptible as each individual
// would be on its own.
//
// Laws:
//  1. It must return if the context is cancelled. It doesn't
//     have to do so immediately, but has to at some point.
//
//  2. It must be safe to interrupt if the state loader is not
//     accepting new loads and has no loaded state units.
//
//  3. It must not leave any orphaned states loaded after
//     returning.
//
//  4. It must be finished when it returns / it must not continue
//     executing backgrounded goroutines after returning.
type Routine func(ctx context.Context, state StateLoader) error

// Run the routine. There are three ways in which this call may terminate:
//
// 1. If the passed in context is cancelled. The passsed in state
//    loader will stop accepting new state loads, and then the routine
//    will be interrupted once all state has been fully unloaded.
//
// 2. If an os signal in Stdsigs is received. The state loader will
//    stop accepting new state loads and the routine will wait the
//    configured amount of time for the specific signal (or wait until
//    done if < 0) while unloading, and then be interrupted if and
//    only if all state was unloaded. If it wasn't, the state loader
//    will resume accepting new state loads.
//
// 3. If the routine returns on its own.
//
// If you want to customize which signals are listened for and their
// configured unload timeouts, use RunSigs to provide your own mapping.
func (f Routine) Run(ctx context.Context) error {
	return f.RunSigs(ctx, Stdsigs)
}

var Stdsigs = map[os.Signal]time.Duration{
	syscall.SIGHUP:  5 * time.Second,
	syscall.SIGINT:  -1,
	syscall.SIGTERM: -1,
}

func listens(sigm map[os.Signal]time.Duration) (<-chan os.Signal, func()) {
	sigs := make(chan os.Signal, 1)
	for k := range sigm {
		signal.Notify(sigs, k)
	}
	return sigs, func() { signal.Stop(sigs) }
}

func withTimeout(ctx context.Context, d time.Duration,
) (context.Context, func()) {
	if d < 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// TODO: SIGINT 3x should force it to kill
func (f Routine) RunSigs(
	ctx context.Context,
	sigm map[os.Signal]time.Duration,
) error {
	var (
		state           = new(State)    // brand new state :)
		sigs, sigCancel = listens(sigm) // listen for configured sigs
		bg              = context.Background()
		fctx, fcancel   = context.WithCancel(bg)
		eg, gctx        = errgroup.WithContext(bg)
	)
	defer sigCancel() // release resources

	// spawn f in background, using the cancellable context
	eg.Go(f.Ap2(state).Ap(fctx))

	for {
	outer:
		state.Accepts(true)

		select {
		case sig := <-sigs:
			// stop accepting state loads
			state.Accepts(false)

			waitCtx, waitCancel := withTimeout(bg, sigm[sig])
			err := state.UnloadWait(waitCtx) // try to unload
			waitCancel()                     // release resources

			if err != nil { // did not unload fast enough
				goto outer // resume accepting state loads
			}

			// we can interrupt tasklet; state is all unloaded
			fcancel()        // interrupt the tasklet
			return eg.Wait() // return its error

		case <-ctx.Done():
			// stop accepting state loads
			state.Accepts(false)

			// wait for loaded state to be unloaded
			if err := state.UnloadWait(bg); err != nil {
				// if this fails theres a bug in UnloadWait
				panic(err)
			}

			// we can interrupt tasklet; state is all unloaded
			fcancel()        // interrupt the tasklet
			return eg.Wait() // return its error

		case <-gctx.Done():
			// the tasklet returned on its own
			state.Accepts(false) // just in case
			fcancel()            // release resources
			return eg.Wait()     // return its error
		}
	}
}

func (f Routine) Binds(bind func(Routine, Routine) Routine,
	gs ...Routine,
) Routine {
	for _, g := range gs {
		f = bind(f, g)
	}
	return f
}

func (f Routine) Seq(gs ...Routine) Routine {
	return Routine.Binds(f, func(x, y Routine) Routine {
		return func(ctx context.Context, state StateLoader) error {
			a := x.Ap2(state)
			b := y.Ap2(state)
			return Tasklet.Seq(a, b)(ctx)
		}
	}, gs...)
}

func (f Routine) Par(gs ...Routine) Routine {
	return Routine.Binds(f, func(x, y Routine) Routine {
		return func(ctx context.Context, state StateLoader) error {
			a := x.Ap2(state).Ap(ctx)
			b := y.Ap2(state).Ap(ctx)
			return Closure.Par(a, b)()
		}
	}, gs...)
}

func (f Routine) Ap(ctx context.Context, state StateLoader) Closure {
	return func() error {
		return f(ctx, state)
	}
}

func (f Routine) Ap1(ctx context.Context) Statelet {
	return func(state StateLoader) error {
		return f(ctx, state)
	}
}

func (f Routine) Ap2(state StateLoader) Tasklet {
	return func(ctx context.Context) error {
		return f(ctx, state)
	}
}

func (f Routine) cond(p Predicate, exit bool) Routine {
	return func(ctx context.Context, state StateLoader) error {
		for {
			if err := f(ctx, state); p(err) == exit {
				return err
			}
		}
	}
}

func (f Routine) While(p Predicate) Routine {
	return Routine.cond(f, p, false)
}

func (f Routine) Until(p Predicate) Routine {
	return Routine.cond(f, p, true)
}

func (f Routine) Mu(mu sync.Locker) Routine {
	return func(ctx context.Context, state StateLoader) error {
		mu.Lock()
		err := f(ctx, state)
		mu.Unlock()
		return err
	}
}

func (f Routine) Wg(wg *sync.WaitGroup) Routine {
	wg.Add(1)
	return func(ctx context.Context, state StateLoader) error {
		defer wg.Done()
		return f(ctx, state)
	}
}
