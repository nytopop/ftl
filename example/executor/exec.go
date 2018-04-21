package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/nytopop/ftl"
	"github.com/nytopop/ftl/fsync"
)

func main() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second*1000,
	)

	//ftl.Debug = true
	e := fsync.NewExecutor(1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := ftl.Routine.Run(e.Routine, ctx)
		wg.Done()
		fmt.Println("exited with", err)
	}()
	time.Sleep(200 * time.Millisecond)

	const (
		nK      = 32
		nT      = 1024
		milReps = 10
	)

	start := time.Now()
	var wgg sync.WaitGroup
	for i := 0; i < nK; i++ {
		wgg.Add(1)
		go func() {
			defer wgg.Done()
			for j := 0; j < nT; j++ {
			here:
				prio := rand.Intn(920)
				var f ftl.Tasklet = func(_ context.Context) error {
					for i := 0; i < milReps*1000000; i++ {
						_ = i * 4
					}
					return nil
				}

				if !e.Add(fsync.Task{
					Priority: prio,
					Action:   f,
					Retry:    ftl.False(),
				}) {
					fmt.Println("hmm")
					goto here
				}
			}
		}()
	}
	wgg.Wait()
	loadT := time.Since(start)
	cancel()
	wg.Wait()
	finT := time.Since(start)

	fmt.Println("finished load in:", loadT, "with n:", nK*nT)
	fmt.Println("finished exec in:", finT)
}
