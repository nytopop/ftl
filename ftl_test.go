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
	"Routine": routineSuite.run,
}

func succ(i *int) func() error {
	return func() error { *i++; return nil }
}
func fail(i *int) func() error {
	return func() error { *i++; return errors.New("") }
}

var testLoader Routine = func(
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
					loaded = state.LoadUnload(
						func(cctx context.Context) error {
							return cctx.Err()
						},
					)
				}()
			} else {
				// don't want to infinite loop the cpu here
				time.Sleep(norm(1))
			}
		}
	}

	return errors.New("unexpectored")
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

		err := Routine.Par(
			testLoader,
			testLoader,
			testLoader,
			testLoader,
			testLoader,
			testLoader,
			testLoader,
		).Run(ctx)

		assert.EqualError(t, err, context.Canceled.Error())
	},
}
