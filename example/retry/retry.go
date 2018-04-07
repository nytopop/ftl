package main

import (
	"fmt"

	"github.com/nytopop/ftl"
)

func prln(args ...interface{}) ftl.Closure {
	return func() error {
		_, err := fmt.Println(args...)
		return err
	}
}

func main() {
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
}
