package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nytopop/ftl"
)

func main() {
	fmt.Println("vim-go")

	g := ftl.Transaction.Seq(
		func(s ftl.STM) error {
			s.Put(1)
			return nil
		},
		func(s ftl.STM) error {
			s.Put(2)
			return nil
		},
		ftl.Transaction.Par(
			func(s ftl.STM) error {
				s.Put("hello")
				s.Put("nvm")
				s.Put("okay")

				// that's an issue...
				// Discard should give back orig
				return nil
			},
			func(s ftl.STM) error {
				s.Put(4)
				return nil
			},
			func(s ftl.STM) error {
				s.Put(5)
				return nil
			},
			func(s ftl.STM) error {
				s.Put(6)
				return nil
			},
		),
	)

	//f(&ftl.STM{})
	stm := ftl.NewSTMv(64)
	//fmt.Println(stm.Atomic(f)())

	f := stm.Atomic(g)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// mhmmm you betcha
	ftl.Closure.Par(
		f, f, f, f, f, f,
		stm.Atomic(func(s ftl.STM) error {
			return nil
		}),
		f, f, f, f, f,
		stm.Atomic(func(s ftl.STM) error {
			s.Put(300303)
			return nil
		}),
		f, f, f, f,
		stm.Atomic(func(s ftl.STM) error {
			s.Put(func() error { return nil })
			return nil
		}),
		f, f, f, f,
		stm.Atomic(func(s ftl.STM) error {
			return errors.New("oboi")
		}),
	).Until(ftl.Done(ctx)).
		Ite(func(err error) bool {
			if err != nil {
				panic(err)
			}
			return false
		},
			func() error { return nil },
			func() error { return nil },
		)()

	// so this right here should _not_ work
	//fmt.Println("ended at", stm.Get())

	/*
		val := new(interface{})
		*val = 123
		fmt.Println("err is", f.Run(val))
		fmt.Println("val is", *val)
	*/
}
