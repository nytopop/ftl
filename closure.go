package ftl

import (
	"sync"

	"golang.org/x/sync/errgroup"
)

// Closure is a function that might fail.
type Closure func() error

func Fail(err error) Closure {
	return func() error { return err }
}

func (f Closure) err(err error) Closure {
	return func() error { return err }
}

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
	return func() error {
		var err error
		if err = f(); err != nil {
			return err
		}
		for i := range gs {
			if err = gs[i](); err != nil {
				break
			}
		}
		return err
	}
}

func (f Closure) Par(gs ...Closure) Closure {
	return func() error {
		var eg errgroup.Group
		eg.Go(f)
		for i := range gs {
			eg.Go(gs[i])
		}
		return eg.Wait()
	}
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

func (f Closure) Ite(p Predicate, g, z Closure) Closure {
	return func() error {
		if err := f(); p(err) {
			return g()
		} else {
			return z()
		}
	}
}

func (f Closure) Mu(mu sync.Locker) Closure {
	return func() error {
		mu.Lock()
		err := f()
		mu.Unlock()
		return err
	}
}

func (f Closure) Once() Closure {
	var once sync.Once
	return func() error {
		var err error
		g := func() { err = f() }
		once.Do(g)
		return err
	}
}
