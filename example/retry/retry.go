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
	// seqA == seqB
	seqA := prln("i").
		Seq(prln("am")).
		Seq(prln("explicitly")).
		Seq(prln("ordered"))

	seqB := ftl.SeqC(
		prln("i"),
		prln("am"),
		prln("explicitly"),
		prln("ordered"),
	)

	prln().Seq(seqA)()
	prln().Seq(seqB)()

	// parA == parB
	parA := ftl.NothingC().
		Par(prln(1)).
		Par(prln(2)).
		Par(prln(3)).
		Par(prln(4))

	parB := ftl.ParC(
		prln(1),
		prln(2),
		prln(3),
		prln(4),
	)

	prln().Seq(parA)()
	prln().Seq(parB)()

	// retryA prints hello world 9 times
	retryA := prln("hello world!").
		Until(ftl.TriesEq(9))

	prln().Seq(retryA)()

	// all intermediate errors are checked, and execution
	// short circuits if an error is returned :)

	// condA == condB
	condA := ftl.TriesEq(2).
		Or(ftl.NotNil())

	condB := ftl.Or(
		ftl.TriesEq(2),
		ftl.NotNil(),
	)

	// retryB == retryC
	retryB := prln().
		Seq(seqB).
		Seq(parB).
		Until(condA)

	retryB()

	// note that the sequence will short circuit if an
	// error is encountered, and the _entire_ sequence
	// will be retried from the start
	retryC := ftl.SeqC(
		prln(),
		seqB,
		parB,
	).Until(condB)

	retryC()
}
