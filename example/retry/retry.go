package main

import (
	"fmt"
	"reflect"
	"time"

	"github.com/nytopop/ftl"
)

func prln(args ...interface{}) ftl.Closure {
	return func() error {
		_, err := fmt.Println(args...)
		return err
	}
}

func main() {
	/*
		seq := ftl.Closure.Seq(
			prln("i"),
			prln("am"),
			prln("explicitly"),
			prln("ordered"),
		)
		prln().Seq(seq)()

		par := ftl.Closure.Par(
			prln(1),
			prln(2),
			prln(3),
			prln(4),
		)

		prln().Seq(par)()

		// retryA prints hello world 9 times
		retryA := prln("hello world!").
			Until(ftl.TriesEq(9))

		prln().Seq(retryA)()

		cond := ftl.Predicate.Or(
			ftl.TriesEq(2),
			ftl.NotNil(),
		)

		// note that the sequence will short circuit if an
		// error is encountered, and the _entire_ sequence
		// will be retried from the start
		retryB := ftl.Closure.Seq(
			prln(),
			seq,
			par,
		).Until(cond)

		retryB()
	*/
	prln("starting until loopr")()

	//var i int
	start := time.Now()
	ftl.Closure.Until(func() error {
		//fmt.Println(i)
		//i++
		return nil
	},
		ftl.TriesEq(2000000),
	)()
	fmt.Println(time.Since(start))

	start = time.Now()
	ftl.Closure.While(func() error {
		//fmt.Println(i)
		//i++
		return nil
	},
		ftl.TriesLte(2000000),
	)()
	fmt.Println(time.Since(start))

	start = time.Now()
	for i := 0; i < 2000000; i++ {
		func() {}()
	}
	fmt.Println(time.Since(start))

	backoffNil := ftl.Predicate.And(
		ftl.Backoff(time.Millisecond, time.Second),
		ftl.TriesLt(12),
		ftl.Nil(),
	)

	f := prln("i can go for a while").While(backoffNil)

	fmt.Println(reflect.TypeOf(f))
	f()
}
