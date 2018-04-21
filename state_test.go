package ftl

import (
	"context"
	"testing"
)

const k = 32768

func BenchmarkState(b *testing.B) {
	b.Run("Load unload()", func(b *testing.B) {
		s := new(State)
		var (
			loaded bool
			unload func()
		)

		for i := 0; i < b.N; i++ {
			s.Accepts(true)
			for j := 0; j < k; j++ {
				loaded, unload = s.Load()
				if loaded {
					unload()
				} else {
					panic(0)
				}
			}
			s.Accepts(false)
			_ = s.UnloadWait(context.Background())
		}
	})

	s := new(State)
	b.Run("LoadUnload", func(b *testing.B) {
		var loaded bool

		for i := 0; i < b.N; i++ {
			s.Accepts(true)
			for j := 0; j < k; j++ {
				loaded = s.LoadUnload(func(_ context.Context) error {
					return nil
				})
				if !loaded {
					panic(0)
				}
			}
			s.Accepts(false)
			_ = s.UnloadWait(context.Background())
		}
	})

	for x := 0; x < 10; x++ {
		s = new(State)
		b.Run("LoadUnload", func(b *testing.B) {
			var loaded bool

			for i := 0; i < b.N; i++ {
				s.Accepts(true)
				for j := 0; j < k; j++ {
					loaded = s.LoadUnload(func(_ context.Context) error {
						return nil
					})
					if !loaded {
						panic(0)
					}
				}
				s.Accepts(false)
				_ = s.UnloadWait(context.Background())
			}
		})
	}
}
