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
// function.
//
// It's a magical contraption that can be composed with other such
// magical contraptions. When executed, the final magical contraption
// is still just as state-safe and interruptible as each individual
// would be on its own.
//
// Expectations:
//
//  1. It returns when the context is cancelled. It doesn't
//     have to do so immediately, but should at some point.
//
//  2. It should be safe to interrupt if the state loader is not
//     accepting new loads and has no loaded state units.
type Routine func(ctx context.Context, state StateLoader) error

// Run the routine. There are two ways in which this call may terminate:
//
// 1. If the passed in context is cancelled. The passsed in state
//    loader will stop accepting new state loads, and then the routine
//    will be interrupted once all state has been fully unloaded.
//
// 2. If the routine returns on its own.
func (f Routine) Run(ctx context.Context) error {
	return f.RunSigM(ctx, nil)
}

// RunSigs runs the routine and listens for os signals. There are
// three ways in which this call may terminate:
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
// configured unload timeouts, use RunSigM to provide your own mapping.
func (f Routine) RunSigs(ctx context.Context) error {
	return f.RunSigM(ctx, Stdsigs)
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

func (f Routine) RunSigM(
	ctx context.Context,
	sigm map[os.Signal]time.Duration,
) error {
	return f.runSigM(ctx, sigm, false)
}

// TODO: SIGINT 3x should force it to kill
func (f Routine) runSigM(
	ctx context.Context,
	sigm map[os.Signal]time.Duration,
	force bool,
) error {
	var (
		state           = new(State)    // brand new state :)
		sigs, sigCancel = listens(sigm) // listen for configured sigs
		bg              = context.Background()
		fctx, fCancel   = context.WithCancel(bg)
		eg, gctx        = errgroup.WithContext(bg)
	)
	defer sigCancel() // release resources

	// spawn f in background, using the cancellable context
	eg.Go(f.Ap(fctx, state))

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

			if force {
				os.Exit(0)
			}

			// we can interrupt tasklet; state is all unloaded
			fCancel()        // interrupt the tasklet
			return eg.Wait() // return its error

		case <-ctx.Done():
			// stop accepting state loads
			state.Accepts(false)

			// wait for loaded state to be unloaded
			var err error
			for {
				err = state.UnloadWait(bg)
				if err == nil {
					break
				}
			}

			if force {
				os.Exit(0)
			}

			// we can interrupt tasklet; state is all unloaded
			fCancel()        // interrupt the tasklet
			return eg.Wait() // return its error

		case <-gctx.Done():
			// the tasklet returned on its own
			state.Accepts(false) // just in case
			fCancel()            // release resources
			if err := eg.Wait(); err != nil {
				if force {
					os.Exit(1)
				}
				return err
			}

			if force {
				os.Exit(0)
			}

			return nil
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
	return func(ctx context.Context, state StateLoader) error {
		fs := make([]Tasklet, len(gs))
		for i := range gs {
			fs[i] = gs[i].Ap2(state)
		}
		return Tasklet.Seq(f.Ap2(state), fs...)(ctx)
	}
}

func (f Routine) Par(gs ...Routine) Routine {
	return func(ctx context.Context, state StateLoader) error {
		var eg errgroup.Group
		eg.Go(f.Ap(ctx, state))
		for _, g := range gs {
			eg.Go(g.Ap(ctx, state))
		}
		return eg.Wait()
	}
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

func (f Routine) Ite(p Predicate, g, z Routine) Routine {
	return func(ctx context.Context, state StateLoader) error {
		if err := f(ctx, state); p(err) {
			return g(ctx, state)
		} else {
			return z(ctx, state)
		}
	}
}

func (f Routine) Mu(mu sync.Locker) Routine {
	return func(ctx context.Context, state StateLoader) error {
		return f.Ap(ctx, state).Mu(mu)()
	}
}

func (f Routine) Once() Routine {
	var once sync.Once
	return func(ctx context.Context, state StateLoader) error {
		var err error
		g := func() { err = f(ctx, state) }
		once.Do(g)
		return err
	}
}
