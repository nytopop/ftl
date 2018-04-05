package ftl

import (
	"sync"

	"golang.org/x/sync/errgroup"
)

type Closure func() error

func NothingC() Closure {
	return func() error { return nil }
}

func (f Closure) until(p Predicate, exit bool) Closure {
	return func() error {
		for {
			err := f()
			if p(err) == exit {
				return err
			}
		}
		panic("inconceivable!")
	}
}

func (f Closure) While(p Predicate) Closure {
	return f.until(p, false)
}

func (f Closure) Until(p Predicate) Closure {
	return f.until(p, true)
}

func (f Closure) apply(bind func(Closure, Closure) Closure,
	gs ...Closure,
) Closure {
	for _, g := range gs {
		f = bind(f, g)
	}
	return f
}

func (f Closure) Seq(g Closure) Closure {
	return func() error {
		if err := f(); err != nil {
			return err
		}
		return g()
	}
}

func SeqC(gs ...Closure) Closure {
	return NothingC().apply((Closure).Seq, gs...)
}

func (f Closure) Par(g Closure) Closure {
	return func() error {
		var eg errgroup.Group
		eg.Go(f)
		eg.Go(g)
		return eg.Wait()
	}
}

func ParC(gs ...Closure) Closure {
	return NothingC().apply((Closure).Par, gs...)
}

func (f Closure) Mu(mu sync.Locker) Closure {
	return func() error {
		mu.Lock()
		err := f()
		mu.Unlock()
		return err
	}
}
