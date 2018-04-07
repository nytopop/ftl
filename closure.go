package ftl

import (
	"sync"

	"golang.org/x/sync/errgroup"
)

// Closure represents a generic function.
type Closure func() error

func (f Closure) Run() error {
	return f()
}

func (f Closure) Binds(bind func(Closure, Closure) Closure,
	gs ...Closure,
) Closure {
	for _, g := range gs {
		f = bind(f, g)
	}
	return f
}

func (f Closure) Seq(gs ...Closure) Closure {
	return Closure.Binds(f, func(x, y Closure) Closure {
		return func() error {
			if err := x(); err != nil {
				return err
			}
			return y()
		}
	}, gs...)
}

func (f Closure) Par(gs ...Closure) Closure {
	return Closure.Binds(f, func(x, y Closure) Closure {
		return func() error {
			var eg errgroup.Group
			eg.Go(x)
			eg.Go(y)
			return eg.Wait()
		}
	}, gs...)
}

func (f Closure) cond(p Predicate, exit bool) Closure {
	return func() error {
		for {
			if err := f(); p(err) == exit {
				return err
			}
		}
	}
}

func (f Closure) While(p Predicate) Closure {
	return Closure.cond(f, p, false)
}

func (f Closure) Until(p Predicate) Closure {
	return Closure.cond(f, p, true)
}

func (f Closure) Mu(mu sync.Locker) Closure {
	return func() error {
		mu.Lock()
		err := f()
		mu.Unlock()
		return err
	}
}

func (f Closure) Wg(wg *sync.WaitGroup) Closure {
	wg.Add(1)
	return func() error {
		defer wg.Done()
		return f()
	}
}
