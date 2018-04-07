package ftl

import "sync"

// Statelet is a stateful function call.
type Statelet func(state StateLoader) error

func (f Statelet) Run() error {
	return f(new(State))
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
	return Statelet.Binds(f, func(x, y Statelet) Statelet {
		return func(state StateLoader) error {
			a := x.Ap(state)
			b := y.Ap(state)
			return Closure.Seq(a, b)()
		}
	}, gs...)
}

func (f Statelet) Par(gs ...Statelet) Statelet {
	return Statelet.Binds(f, func(x, y Statelet) Statelet {
		return func(state StateLoader) error {
			a := x.Ap(state)
			b := y.Ap(state)
			return Closure.Par(a, b)()
		}
	}, gs...)
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

func (f Statelet) Mu(mu sync.Locker) Statelet {
	return func(state StateLoader) error {
		mu.Lock()
		err := f(state)
		mu.Unlock()
		return err
	}
}

func (f Statelet) Wg(wg *sync.WaitGroup) Statelet {
	wg.Add(1)
	return func(state StateLoader) error {
		defer wg.Done()
		return f(state)
	}
}
