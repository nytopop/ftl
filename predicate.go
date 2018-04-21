package ftl

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type Predicate func(error) bool

func (p Predicate) Run(err error) bool {
	return p(err)
}

func (p Predicate) Binds(bind func(Predicate, Predicate) Predicate,
	gs ...Predicate,
) Predicate {
	for _, g := range gs {
		p = bind(p, g)
	}
	return p
}

func (f Predicate) And(gs ...Predicate) Predicate {
	return Predicate.Binds(f, func(x, y Predicate) Predicate {
		return func(err error) bool {
			return x(err) && y(err)
		}
	}, gs...)
}

func (f Predicate) Or(gs ...Predicate) Predicate {
	return Predicate.Binds(f, func(x, y Predicate) Predicate {
		return func(err error) bool {
			return x(err) || y(err)
		}
	}, gs...)
}

func (p Predicate) Not() Predicate {
	return func(err error) bool {
		return !p(err)
	}
}

func True() Predicate {
	return func(_ error) bool { return true }
}

func False() Predicate {
	return func(_ error) bool { return false }
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

func Backoff(start, ceil time.Duration) Predicate {
	var (
		i     int
		sleep = start
	)
	return func(_ error) bool {
		i++
		if i == 1 {
			return true
		}

		time.Sleep(sleep)

		sleep = sleep * 2
		if sleep > ceil {
			sleep = ceil
		}

		return true
	}
}
