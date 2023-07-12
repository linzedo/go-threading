package threading

import (
	"sync"
	"testing"
	"time"
)

const (
	Concurrency = 1
	executeTime = 1000
)

func BenchmarkCurrent1Reuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithReuse(Concurrency)
	}
}

func BenchmarkCurrent2Reuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithReuse(Concurrency * 2)
	}
}

func BenchmarkCurrent10Reuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithReuse(Concurrency * 10)
	}
}
func BenchmarkCurrent10KReuse(b *testing.B) {
	//var memStats runtime.MemStats
	//runtime.ReadMemStats(&memStats)
	for i := 0; i < b.N; i++ {
		WithReuse(Concurrency * 10000)
	}
	//fmt.Printf("Allocated memory: %d bytes\n", memStats.Alloc)
	//fmt.Printf("Total memory allocated and not yet freed: %d bytes\n", memStats.TotalAlloc)
	//fmt.Printf("Heap memory allocated: %d bytes\n", memStats.HeapAlloc)
	//fmt.Printf("Total heap memory obtained from system: %d bytes\n", memStats.HeapSys)
}

func BenchmarkCurrent1NotReuse1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse1(Concurrency * 1)
	}
}
func BenchmarkCurrent2NotReuse1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse1(Concurrency * 2)
	}
}
func BenchmarkCurrent10NotReuse1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse1(Concurrency * 10)
	}
}
func BenchmarkCurrent10kNotReuse1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse1(Concurrency * 10000)
	}
}

func BenchmarkCurrent1NotReuse2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse2(Concurrency * 1)
	}
}
func BenchmarkCurrent2NotReuse2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse2(Concurrency * 2)
	}
}
func BenchmarkCurrent10NotReuse2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse2(Concurrency * 10)
	}
}
func BenchmarkCurrent10kNotReuse2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse2(Concurrency * 10000)
	}
}

func WithReuse(c int) {
	gs := New(Config{
		GoCount: c,
		Wait:    true,
	})
	for j := 0; j < c; j++ {
		_ = gs.Go(func() error {
			time.Sleep(time.Microsecond * executeTime)
			return nil
		})
	}
	_ = gs.Wait()
}

func WithoutReuse1(c int) {
	gs := New(Config{
		GoCount:  c,
		Wait:     true,
		NotReuse: true,
	})
	for j := 0; j < c; j++ {
		_ = gs.Go(func() error {
			time.Sleep(time.Microsecond * executeTime)
			return nil
		})
	}
	_ = gs.Wait()
}

func WithoutReuse2(c int) {
	wp := sync.WaitGroup{}
	for j := 0; j < c; j++ {
		wp.Add(1)
		go func() {
			time.Sleep(time.Microsecond * executeTime)
			wp.Done()
		}()
	}
	wp.Wait()
}
