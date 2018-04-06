package ftl

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Tasklet represents a stateless interruptible function.
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

func (f Tasklet) bindWith(bind func(Tasklet, Tasklet) Tasklet,
	gs ...Tasklet,
) Tasklet {
	x := f
	for _, g := range gs {
		x = bind(x, g)
	}
	return x
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
	return NothingT().bindWith((Tasklet).Seq, gs...)
}

func (f Tasklet) Ap(ctx context.Context) Closure {
	return func() error {
		return f(ctx)
	}
}

func (f Tasklet) Par(g Tasklet) Tasklet {
	return func(ctx context.Context) error {
		eg, taskCtx := errgroup.WithContext(ctx)
		eg.Go(f.Ap(taskCtx))
		eg.Go(g.Ap(taskCtx))
		return eg.Wait()
	}
}

func ParT(gs ...Tasklet) Tasklet {
	return NothingT().bindWith((Tasklet).Par, gs...)
}

func (f Tasklet) Mu(mu sync.Locker) Tasklet {
	return func(ctx context.Context) error {
		mu.Lock()
		err := f(ctx)
		mu.Unlock()
		return err
	}
}

func (f Tasklet) Wg(wg *sync.WaitGroup) Tasklet {
	wg.Add(1)
	return func(ctx context.Context) error {
		defer wg.Done()
		return f(ctx)
	}
}

func (f Tasklet) SeqN(n int, g Tasklet) Tasklet {
	for i := 1; i < n; i++ {
		f = f.Seq(g)
	}
	return f
}

func (f Tasklet) ParN(n int, g Tasklet) Tasklet {
	for i := 1; i < n; i++ {
		f = f.Par(g)
	}
	return f
}
