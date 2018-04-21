//go:generate genny -in=ftm.go -out=gen-ftm.go gen "Val=BUILTINS"
package ftm

import (
	"sync"

	"github.com/cheekybits/genny/generic"
	"github.com/nytopop/ftl"
)

type Val generic.Type

// FTMVal exposes a thread-safe, mutable view into any
// value of type Val.
//
// Internally, nested mutexes are used instead of the
// usual retry mechanism.
type FTMVal interface {
	// Mut returns a pointer to a safely mutable Val.
	//
	// This reference must not escape the transaction
	// in which it was created.
	Mut() *Val

	// Discard any changes since the last checkpoint.
	Discard()

	open() FTMVal
	atomic(TxVal) ftl.Closure
}

// FTMValV exposes an opaque view into any value of
// type Val. Reading or writing the value requires
// the use of a call to atomic.
type FTMValV interface {
	Atomic(TxVal) ftl.Closure
}

type ftmVal struct {
	orig Val
	mut  *Val
	mu   sync.Mutex
}

func newFTMVal(v Val) *ftmVal {
	return &ftmVal{
		orig: v,
		mut:  &v,
	}
}

func NewFTMValV(v Val) FTMValV {
	return newFTMVal(v)
}

func (s *ftmVal) Mut() *Val {
	return s.mut
}

func (s *ftmVal) Discard() {
	*s.mut = s.orig
}

func (s *ftmVal) open() FTMVal {
	return &ftmVal{
		orig: *s.mut,
		mut:  s.mut,
	}
}

func (s *ftmVal) atomic(tx TxVal) ftl.Closure {
	return func() error {
		s.mu.Lock()
		err := tx(s)
		s.mu.Unlock()
		return err
	}
}

func (s *ftmVal) Atomic(tx TxVal) ftl.Closure {
	return func() error {
		s.mu.Lock()
		err := tx(s.open())
		s.mu.Unlock()
		return err
	}
}

type TxVal func(FTMVal) error

func (f TxVal) Run(v Val) error {
	return f(newFTMVal(v))
}

func (f TxVal) Binds(bind func(TxVal, TxVal) TxVal,
	gs ...TxVal,
) TxVal {
	for _, g := range gs {
		f = bind(f, g)
	}
	return f
}

func (f TxVal) Checkpoint(gs ...TxVal) TxVal {
	return func(s FTMVal) error {
		var (
			inner = s.open()
			g     = inner.atomic(f)
			fs    = make([]ftl.Closure, len(gs))
		)
		for i := range gs {
			fs[i] = inner.atomic(gs[i])
		}
		return ftl.Closure.Seq(g, fs...)()
	}
}

func (f TxVal) Seq(gs ...TxVal) TxVal {
	return func(s FTMVal) error {
		var (
			fs = make([]ftl.Closure, len(gs))
			g  = f.ap(s)
		)
		for i := range gs {
			fs[i] = gs[i].ap(s)
		}
		return ftl.Closure.Seq(g, fs...)()
	}
}

// ap is unexported because `f.ap(s).Par(g)()` is
// very unsafe.
func (f TxVal) ap(s FTMVal) ftl.Closure {
	return func() error {
		return f(s)
	}
}

func (f TxVal) cond(p ftl.Predicate, exit bool) TxVal {
	return func(s FTMVal) error {
		for {
			if err := f(s); p(err) == exit {
				return err
			}
		}
	}
}

func (f TxVal) While(p ftl.Predicate) TxVal {
	return TxVal.cond(f, p, false)
}

func (f TxVal) Until(p ftl.Predicate) TxVal {
	return TxVal.cond(f, p, true)
}

func (f TxVal) Ite(p ftl.Predicate, g, z TxVal) TxVal {
	return func(s FTMVal) error {
		if err := f(s); p(err) {
			return g(s)
		} else {
			return z(s)
		}
	}
}

func (f TxVal) Mu(mu sync.Locker) TxVal {
	return func(s FTMVal) error {
		return f.ap(s).Mu(mu)()
	}
}

func (f TxVal) Once() TxVal {
	var once sync.Once
	return func(s FTMVal) error {
		var err error
		g := func() { err = f(s) }
		once.Do(g)
		return err
	}
}
