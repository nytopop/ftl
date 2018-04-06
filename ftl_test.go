package ftl

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"

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
	"Predicate": predicateSuite.run,
	"Closure":   closureSuite.run,
	"Tasklet":   taskletSuite.run,
	"Routine":   routineSuite.run,
}

func succ(i *int) func() error {
	return func() error { *i++; return nil }
}
func fail(i *int) func() error {
	return func() error { *i++; return errors.New("") }
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

func testLoader(t *testing.T,
	ctx context.Context,
	state StateLoader,
) error {
	ticker := time.NewTicker(time.Second / 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

			// this is simulating incoming connections at cpu speed
		case <-ticker.C:
			// try to load state
			loaded, unload := state.Load()

			// if allowed to, actually load it
			if loaded {
				go func() {
					// unload state
					unload()

					// load some state for the other way
					loaded = state.LoadUnload(NothingT())
					assert.True(t, loaded)
				}()
			} else {
				// don't want to infinite loop the cpu here
				time.Sleep(norm(1))
			}
		}
	}

	return errors.New("unexpectored")
}

func mk(t *testing.T, n int) []Routine {
	fs := make([]Routine, n)
	for i := 0; i < n; i++ {
		fs[i] = func(ctx context.Context, state StateLoader) error {
			return testLoader(t, ctx, state)
		}
	}
	return fs
}

func norm(x int64) time.Duration {
	r := rand.NormFloat64()
	if r < 0 {
		r = r * (-1)
	}
	return time.Duration(int64(float64(x)*r)) * time.Millisecond
}

var routineSuite = suite{
	"BindR": func(t *testing.T) {
		rand.Seed(time.Now().UnixNano())

		// set debug so loads/unloads and current # of states are printed
		// also, printing bottlenecks the whole thing at state.Load, unload
		// which is a v. good thing for memory constraints :)
		Debug = true

		// configure a ctx
		ctx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second, // run for 3 secs, unless ^C
		)
		defer cancel()

		err := BindR(
			mk(t, 7)...,
		).Run(ctx)

		assert.EqualError(t, err, context.Canceled.Error())
	},
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
