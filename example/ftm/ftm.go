package main

import (
	"context"
	"errors"
	"time"

	"github.com/nytopop/ftl"
	"github.com/nytopop/ftl/ftm"
)

func main() {
	g := ftm.TxInt.Seq(
		func(s ftm.FTMInt) error {
			*(s.Mut())++
			return nil
		},
		func(s ftm.FTMInt) error {
			*(s.Mut())++
			return nil
		},
		ftm.TxInt.Checkpoint(
			func(s ftm.FTMInt) error {
				*(s.Mut())++
				return nil
			},
			func(s ftm.FTMInt) error {
				*(s.Mut())++
				return nil
			},
			func(s ftm.FTMInt) error {
				*(s.Mut())++
				return nil
			},
		),
	)

	//f(&ftl.FTMInt{})
	stm := ftm.NewFTMIntV(64)
	//fmt.Println(stm.Atomic(f)())

	f := stm.Atomic(g)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// mhmmm you betcha
	ftl.Closure.Par(
		f, f, f, f, f, f,
		stm.Atomic(func(s ftm.FTMInt) error {
			return nil
		}),
		f, f, f, f, f,
		stm.Atomic(func(s ftm.FTMInt) error {
			*(s.Mut()) = 300303
			return nil
		}),
		f, f, f, f,
		f, f, f, f,
		stm.Atomic(func(s ftm.FTMInt) error {
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
