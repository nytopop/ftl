package ftl

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Tasklet is an interruptible function.
//
// Expectations:
//  1. It returns when the context is cancelled. It doesn't
//     have to do so immediately, but should at some point.
type Tasklet func(context.Context) error

func (f Tasklet) Run() error {
	return f(context.Background())
}

func (f Tasklet) Binds(bind func(Tasklet, Tasklet) Tasklet,
	gs ...Tasklet,
) Tasklet {
	x := f
	for _, g := range gs {
		x = bind(x, g)
	}
	return x
}

var retErr Tasklet = func(ctx context.Context) error {
	return ctx.Err()
}

func (f Tasklet) Seq(gs ...Tasklet) Tasklet {
	return func(ctx context.Context) error {
		var err error
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = f(ctx); err != nil {
			return err
		}
		for _, g := range gs {
			if err = ctx.Err(); err != nil {
				return err
			}
			if err = g(ctx); err != nil {
				return err
			}
		}
		return err
	}
}

func (f Tasklet) Par(gs ...Tasklet) Tasklet {
	return func(ctx context.Context) error {
		eg, taskCtx := errgroup.WithContext(ctx)
		eg.Go(f.Ap(taskCtx))
		for _, g := range gs {
			eg.Go(g.Ap(taskCtx))
		}
		return eg.Wait()
	}
}

func (f Tasklet) Ap(ctx context.Context) Closure {
	return func() error {
		return f(ctx)
	}
}

func (f Tasklet) cond(p Predicate, exit bool) Tasklet {
	return func(ctx context.Context) error {
		for {
			if err := f(ctx); p(err) == exit {
				return err
			}
		}
	}
}

func (f Tasklet) While(p Predicate) Tasklet {
	return Tasklet.cond(f, p, false)
}

func (f Tasklet) Until(p Predicate) Tasklet {
	return Tasklet.cond(f, p, true)
}

func (f Tasklet) Ite(p Predicate, g, z Tasklet) Tasklet {
	return func(ctx context.Context) error {
		if err := f(ctx); p(err) {
			return g(ctx)
		} else {
			return z(ctx)
		}
	}
}

func (f Tasklet) Mu(mu sync.Locker) Tasklet {
	return func(ctx context.Context) error {
		return f.Ap(ctx).Mu(mu)()
	}
}

func (f Tasklet) Once() Tasklet {
	var once sync.Once
	return func(ctx context.Context) error {
		var err error
		g := func() { err = f(ctx) }
		once.Do(g)
		return err
	}
}
