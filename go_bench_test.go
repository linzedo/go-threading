package threading

import (
	"github.com/letsfire/factory"
	"github.com/panjf2000/ants"
	"sync"
	"testing"
	"time"
)

const (
	Concurrency = 10
	executeTime = 10000
	goCount     = 10000
)

func Job() {
	time.Sleep(time.Millisecond * 10)
	//for i := 0; i < 10000; i++ {
	//	i++
	//}
}
func TestMain(m *testing.M) {
	m.Run()
}
func BenchmarkCurrent1Reuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithReuse(Concurrency)
	}
}

func BenchmarkCurrent10Reuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithReuse(Concurrency * 10)
	}
}

func BenchmarkCurrent100Reuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithReuse(Concurrency * 100)
	}
}

func BenchmarkCurrent1kReuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithReuse(Concurrency * 1000)
	}
}

func BenchmarkCurrent10kReuse(b *testing.B) {
	StartPool(SetMaxIdleWorkerDuration(time.Second*30), SetMinWorkCount(goCount))
	for i := 0; i < b.N; i++ {
		gs := New(Config{
			GoCount: goCount,
		})
		for j := 0; j < goCount; j++ {
			_ = gs.Go(func() error {
				Job()
				return nil
			})
		}
		_ = gs.Wait()
	}
}

//factory
func BenchmarkCurrent10kReuseFactory(b *testing.B) {
	var master = factory.NewMaster(goCount, goCount)
	for i := 0; i < b.N; i++ {
		var line1 = master.AddLine(func(args interface{}) {
			Job()
		})
		for j := 0; j < goCount; j++ {
			line1.Submit(j)
		}
		line1.Wait()
	}

}

func BenchmarkCurrent10kReuseAnt(b *testing.B) {
	var p, _ = ants.NewPool(goCount)
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < goCount; j++ {
			wg.Add(1)
			_ = p.Submit(func() {
				Job()
				wg.Done()
			})
		}
		wg.Wait()
		p.Release()
	}

}

func BenchmarkCurrent1NotReuse1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse1(Concurrency * 1)
	}
}
func BenchmarkCurrent10NotReuse1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse1(Concurrency * 10)
	}
}
func BenchmarkCurrent100NotReuse1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse1(Concurrency * 100)
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
func BenchmarkCurrent10NotReuse2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse2(Concurrency * 10)
	}
}
func BenchmarkCurrent100NotReuse2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WithoutReuse2(Concurrency * 100)
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
			Job()
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
			Job()
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
			Job()
			wp.Done()
		}()
	}
	wp.Wait()
}
