package ftl

import (
	"context"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Statelet is a stateful function call.
type Statelet func(state StateLoader) error

func (f Statelet) Run(ctx context.Context) {
	Statelet.RunSigM(f, ctx, nil)
}

func (f Statelet) RunSigs(ctx context.Context) {
	Statelet.RunSigM(f, ctx, Stdsigs)
}

func (f Statelet) RunSigM(
	ctx context.Context,
	sigm map[os.Signal]time.Duration,
) {
	_ = Routine.runSigM(
		func(_ context.Context, state StateLoader) error {
			return f(state)
		},
		ctx,
		sigm,
		true,
	)
}

func (f Statelet) Binds(bind func(x, y Statelet) Statelet,
	gs ...Statelet,
) Statelet {
	for _, g := range gs {
		f = bind(f, g)
	}
	return f
}

func (f Statelet) Seq(gs ...Statelet) Statelet {
	return func(state StateLoader) error {
		var err error
		if err = f(state); err != nil {
			return err
		}
		for _, g := range gs {
			if err = g(state); err != nil {
				return err
			}
		}
		return err
	}
}

func (f Statelet) Par(gs ...Statelet) Statelet {
	return func(state StateLoader) error {
		var eg errgroup.Group
		eg.Go(f.Ap(state))
		for _, g := range gs {
			eg.Go(g.Ap(state))
		}
		return eg.Wait()
	}
}

func (f Statelet) Ap(state StateLoader) Closure {
	return func() error {
		return f(state)
	}
}

func (f Statelet) cond(p Predicate, exit bool) Statelet {
	return func(state StateLoader) error {
		for {
			if err := f(state); p(err) == exit {
				return err
			}
		}
	}
}

func (f Statelet) While(p Predicate) Statelet {
	return Statelet.cond(f, p, false)
}

func (f Statelet) Until(p Predicate) Statelet {
	return Statelet.cond(f, p, true)
}

func (f Statelet) Ite(p Predicate, g, z Statelet) Statelet {
	return func(state StateLoader) error {
		if err := f(state); p(err) {
			return g(state)
		} else {
			return z(state)
		}
	}
}

func (f Statelet) Mu(mu sync.Locker) Statelet {
	return func(state StateLoader) error {
		return f.Ap(state).Mu(mu)()
	}
}

func (f Statelet) Once() Statelet {
	var once sync.Once
	return func(state StateLoader) error {
		var err error
		g := func() { err = f(state) }
		once.Do(g)
		return err
	}
}
