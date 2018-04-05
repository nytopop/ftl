package ftl

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

type Tasklet func(context.Context) error

func NothingT() Tasklet {
	return func(ctx context.Context) error {
		return ctx.Err()
	}
}

func (f Tasklet) until(p Predicate, exit bool) Tasklet {
	return func(ctx context.Context) error {
		for {
			err := f(ctx)
			if p(err) == exit {
				return err
			}
		}
		panic("inconceivable!")
	}
}

func (f Tasklet) While(p Predicate) Tasklet {
	return f.until(p, false)
}

func (f Tasklet) Until(p Predicate) Tasklet {
	return f.until(p, true)
}

func (f Tasklet) apply(bind func(Tasklet, Tasklet) Tasklet,
	gs ...Tasklet,
) Tasklet {
	for _, g := range gs {
		f = bind(f, g)
	}
	return f
}

func (f Tasklet) Seq(g Tasklet) Tasklet {
	return func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := f(ctx); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return g(ctx)
	}
}

func SeqT(gs ...Tasklet) Tasklet {
	return NothingT().apply((Tasklet).Seq, gs...)
}

func (f Tasklet) lift(ctx context.Context) Closure {
	return func() error {
		return f(ctx)
	}
}

func (f Tasklet) Par(g Tasklet) Tasklet {
	return func(ctx context.Context) error {
		eg, taskCtx := errgroup.WithContext(ctx)
		eg.Go(f.lift(taskCtx))
		eg.Go(g.lift(taskCtx))
		return eg.Wait()
	}
}

func ParT(gs ...Tasklet) Tasklet {
	return NothingT().apply((Tasklet).Par, gs...)
}

func (f Tasklet) Mu(mu sync.Locker) Tasklet {
	return func(ctx context.Context) error {
		mu.Lock()
		err := f(ctx)
		mu.Unlock()
		return err
	}
}
