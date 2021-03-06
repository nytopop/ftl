package main

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/nytopop/ftl"
)

func service(state ftl.StateLoader) error {
	// load state as fast as possible
	for {
		// this is simulating incoming connections at cpu speed
		// try to load state
		loaded, unload := state.Load()

		// if allowed to, actually load it
		if loaded {
			go func() {
				// simulating some time passing
				// norm distrib centered on millis
				//
				// it's nice because the outliers end
				// up having a greater effect on the service,
				// much like real traffic
				//time.Sleep(norm(4000))

				// unload state
				unload()

				// load some state for the other way
				// interesting how memory only spikes once
				// the stack is unfolded during unloading,
				// as its an in place mutation.
				if rand.Intn(15) == 1 {
					var i int
					_ = state.LoadUnload(
						func(cctx context.Context) error {
							if i > 0 {
								panic("shit")
							}

							i++
							return cctx.Err()
						},
					)
				}
			}()
		} else {
			// don't want to infinite loop the cpu here
			time.Sleep(norm(1))
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

func main() {
	rand.Seed(time.Now().UnixNano())

	// set debug so loads/unloads and current # of states are printed
	// also, printing bottlenecks the whole thing at state.Load, unload
	// which is a v. good thing for memory constraints :)
	ftl.Debug = true

	// configure a ctx
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second, // run for 10 secs, unless ^C
	)
	defer cancel()

	//sigs := map[os.Signal]time.Duration{
	//	syscall.SIGINT: 2 * time.Second,
	//}

	// run 9 stateful services, with varying load
	// durations and very large numbers of loads. if this
	// works correctly, everything is perfectly synchronized,
	// concurrently (!)
	//
	// :) stabilizes at around 630,000 concurrent state loads
	// on my machine (while debug is enabled - if it's off then
	// it goes into redonkulous amounts of memory in goroutines)
	ftl.Statelet.Par(
		service,
		service,
		service,
		service,
		service,
		service,
		service,
		service,
		service,
	).RunSigs(ctx)
}
