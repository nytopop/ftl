package fsync

import (
	"context"
	"sort"
	"testing"

	"github.com/emirpasic/gods/trees/redblacktree"
)

const gk = 1024

func BenchmarkHeap(b *testing.B) {
	//ts := make([]Task, 32)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pq := newQueue(0)
		b.StartTimer()

		for j := 0; j < gk; j++ {
			pq.push(Task{
				Priority: j,
				Action: func(_ context.Context) error {
					for x := 0; x < 10000; x++ {
						_ = x
					}
					return nil
				},
			})
		}

		ts := pq.flush()
		_ = ts
	}
}

func BenchmarkHeapflushB(b *testing.B) {
	b.StopTimer()
	ts := make([]Task, gk)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pq := newQueue(0)
		b.StartTimer()

		for j := 0; j < gk; j++ {
			pq.push(Task{
				Priority: j,
				Action: func(_ context.Context) error {
					for x := 0; x < 10000; x++ {
						_ = x
					}
					return nil
				},
			})
		}

		n := pq.flushB(ts)
		_ = n
	}
}

func BenchmarkTree(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tree := redblacktree.NewWithIntComparator()
		b.StartTimer()

		for j := 0; j < gk; j++ {
			tree.Put(j, Task{
				Priority: j,
				Action: func(_ context.Context) error {
					for x := 0; x < 10000; x++ {
						_ = x
					}
					return nil
				},
			})
		}

		is := tree.Values()
		tree.Clear()
		// can drop mutex here
		ts := make([]Task, len(is))
		for j := range is {
			ts[j] = is[j].(Task)
		}
	}
}

func BenchmarkSortedSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		slice := make(taskQueue, 0)
		b.StartTimer()

		for j := 0; j < gk; j++ {
			slice = append(slice, Task{
				Priority: j,
				Action: func(_ context.Context) error {
					for x := 0; x < 10000; x++ {
						_ = x
					}
					return nil
				},
			})
		}

		// can drop mutex here
		// wow. so this is twice as fast as the heap :|
		sort.Sort(&slice)
		ts := make([]Task, len(slice))
		copy(ts, slice)
		slice = nil
	}
}
