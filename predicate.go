package ftl

import (
	"context"

	"github.com/pkg/errors"
)

type Predicate func(error) bool

func (p Predicate) bindWith(bind func(Predicate, Predicate) Predicate,
	gs ...Predicate,
) Predicate {
	for _, g := range gs {
		p = bind(p, g)
	}
	return p
}

func True() Predicate {
	return func(_ error) bool { return true }
}

func False() Predicate {
	return func(_ error) bool { return false }
}

func (f Predicate) And(g Predicate) Predicate {
	return func(err error) bool {
		fr := f(err)
		gr := g(err)
		return fr && gr
	}
}

func And(gs ...Predicate) Predicate {
	return True().bindWith((Predicate).And, gs...)
}

func (f Predicate) Or(g Predicate) Predicate {
	return func(err error) bool {
		fr := f(err)
		gr := g(err)
		return fr || gr
	}
}

func Or(gs ...Predicate) Predicate {
	return False().bindWith((Predicate).Or, gs...)
}

func (p Predicate) Not() Predicate {
	return func(err error) bool {
		return !p(err)
	}
}

func TriesEq(n int) Predicate {
	var tries int
	return func(_ error) bool {
		tries++
		return tries == n
	}
}

func TriesLt(n int) Predicate {
	var tries int
	return func(_ error) bool {
		tries++
		return tries < n
	}
}

func TriesLte(n int) Predicate {
	var tries int
	return func(_ error) bool {
		tries++
		return tries <= n
	}
}

func TriesGt(n int) Predicate {
	var tries int
	return func(_ error) bool {
		tries++
		return tries > n
	}
}

func TriesGte(n int) Predicate {
	var tries int
	return func(_ error) bool {
		tries++
		return tries >= n
	}
}

func Error(err error) Predicate {
	return func(er error) bool {
		return errors.Cause(er) == err
	}
}

func Nil() Predicate {
	return func(err error) bool {
		return err == nil
	}
}

func NotNil() Predicate {
	return func(err error) bool {
		return err != nil
	}
}

func Done(ctx context.Context) Predicate {
	return func(_ error) bool {
		return ctx.Err() != nil
	}
}

func NotDone(ctx context.Context) Predicate {
	return func(_ error) bool {
		return ctx.Err() == nil
	}
}
