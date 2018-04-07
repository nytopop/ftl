package ftl

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Tasklet represents a stateless interruptible function.
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

func (f Tasklet) Seq(gs ...Tasklet) Tasklet {
	return Tasklet.Binds(f, func(x, y Tasklet) Tasklet {
		return func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := x(ctx); err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			return y(ctx)
		}
	}, gs...)
}

func (f Tasklet) Par(gs ...Tasklet) Tasklet {
	return Tasklet.Binds(f, func(x, y Tasklet) Tasklet {
		return func(ctx context.Context) error {
			eg, taskCtx := errgroup.WithContext(ctx)
			eg.Go(x.Ap(taskCtx))
			eg.Go(y.Ap(taskCtx))
			return eg.Wait()
		}
	}, gs...)
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
