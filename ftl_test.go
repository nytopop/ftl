package ftl

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type suite map[string]func(t *testing.T)

func (s suite) run(t *testing.T) {
	for name, test := range s {
		t.Run(name, test)
	}
}

func TestFTL(t *testing.T) { tests.run(t) }

var tests = suite{
	"Closure":   closureSuite.run,
	"Tasklet":   taskletSuite.run,
	"Predicate": predicateSuite.run,
}

func succ(i *int) func() error {
	return func() error { *i++; return nil }
}
func fail(i *int) func() error {
	return func() error { *i++; return errors.New("") }
}

var closureSuite = suite{
	"NothingC": func(t *testing.T) {
		err := NothingC()()
		assert.NoError(t, err)
	},

	// TODO
	"While": suite{}.run,
	"Until": suite{}.run,

	"SeqC": suite{
		"short circuits on failure": func(t *testing.T) {
			var i, j int
			f := fail(&i)
			g := succ(&j)

			err := SeqC(g, f, g)()
			assert.Error(t, err)
			assert.Equal(t, 1, i)
			assert.Equal(t, 1, j)
		},
	}.run,
	"ParC": suite{
		"all finish executing": func(t *testing.T) {
			var i, j int
			f := fail(&i)
			g := succ(&j)

			err := ParC(f, g)()
			assert.Error(t, err)
			assert.Equal(t, i, 1)
			assert.Equal(t, j, 1)
		},
	}.run,

	// TODO
	"Mu": suite{}.run,
}

var taskletSuite = suite{
	"NothingT": suite{
		"returns nil when open": func(t *testing.T) {
			f := NothingT()
			assert.NoError(t, f(context.Background()))
		},
		"returns err when closed": func(t *testing.T) {
			f := NothingT()

			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			cancel()

			assert.Error(t, f(ctx))
			assert.EqualError(t, f(ctx), context.Canceled.Error())
		},
	}.run,

	// TODO...
	"While": suite{}.run,
	"Until": suite{}.run,
	"SeqT":  suite{}.run,
	"ParT":  suite{}.run,
	"Mu":    suite{}.run,
}

var predicateSuite = suite{
	"True": func(t *testing.T) {
		assert.True(t, True()(nil))
		assert.True(t, True()(errors.New("ok")))
	},
	"False": func(t *testing.T) {
		assert.False(t, False()(nil))
		assert.False(t, False()(errors.New("ok")))
	},

	// TODO(no time for that :p)
	"And": suite{}.run,
	"Or": func(t *testing.T) {
		assert.True(t, Or(True())(nil))
		assert.True(t, Or(True(), True())(nil))
		assert.True(t, Or(True(), False())(nil))
		assert.True(t, Or(False(), True())(nil))

		assert.False(t, Or(False())(nil))
		assert.False(t, Or(False(), False())(nil))
	},
	"Not":      suite{}.run,
	"TriesEq":  suite{}.run,
	"TriesLt":  suite{}.run,
	"TriesLte": suite{}.run,
	"TriesGt":  suite{}.run,
	"TriesGte": suite{}.run,
	"Error":    suite{}.run,
	"Nil":      suite{}.run,
	"NotNil":   suite{}.run,
	"Done":     suite{}.run,
	"NotDone":  suite{}.run,
}

/*
func TestFTL(t *testing.T) {
	suite{
		"asdf": nil,
	}.run(t)

	erra := errors.New("")

	var counter int
	f := func() error {
		counter++

		if counter >= 6 {
			return erra
		}

		return nil
	}

	counter = 0
	g := Until(f, Tries(4))
	err := g()
	assert.NoError(t, err)
	assert.Equal(t, 4, counter)

	counter = 0
	g = Until(f, Or(Tries(8), NotNil()))
	err = g()
	assert.Error(t, err)
	assert.Equal(t, 6, counter)

	counter = 0
	g = Until(f, Or(Tries(8), Error(erra)))
	err = g()
	assert.Error(t, err)
	assert.Equal(t, 6, counter)

	counter = 0
	g = Until(f, Or(Error(erra), Tries(8)))
	err = g()
	assert.Error(t, err)
	assert.Equal(t, 6, counter)
}
*/
