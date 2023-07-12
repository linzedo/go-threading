package threading

import (
	"testing"
)

func TestMain(m *testing.M) {
	//goleak.VerifyTestMain(m)
	m.Run()
}

//
//func TestGoWait(t *testing.T) {
//	gs := New(Config{
//		Limit:    5,
//		GoCount:  4,
//		Wait:     true,
//		NotReuse: true,
//	})
//	_ = gs.Go(func() error {
//		gs.Err(errors.New("error1"))
//		return nil
//	})
//	_ = gs.Go(func() error {
//		return errors.New("error2")
//	})
//	_ = gs.Go(func() error {
//		panic("panic3")
//		return nil
//	})
//	_ = gs.Go(func() error {
//		panic("panic4")
//		return nil
//	})
//	fmt.Println("over:", gs.Go(func() error {
//		panic("panic5")
//		return nil
//	}))
//
//	err := gs.Wait()
//	fmt.Println(err)
//
//}
//
//func TestWithoutWait(t *testing.T) {
//	for i := 0; i < 1000; i++ {
//		gs := New(Config{
//			Limit:    5,
//			GoCount:  4,
//			Wait:     false,
//			NotReuse: true,
//		})
//		_ = gs.Go(func() error {
//			panic("panic4")
//			return nil
//		})
//		_ = gs.Go(func() error {
//			gs.Err(errors.New("error1"))
//			return nil
//		})
//		_ = gs.Go(func() error {
//			panic("panic5")
//			return nil
//		})
//		_ = gs.Go(func() error {
//			gs.Err(errors.New("record"))
//			return nil
//		})
//		err := gs.Wait()
//		fmt.Println(err)
//	}
//	time.Sleep(time.Second)
//}
//
//func TestGoSyncReuse(t *testing.T) {
//	for i := 0; i < 1000; i++ {
//		gs := New(Config{
//			Limit:   5,
//			GoCount: 4,
//			Wait:    true,
//		})
//		_ = gs.Go(func() error {
//			panic("panic4")
//			return nil
//		})
//		_ = gs.Go(func() error {
//			gs.Err(errors.New("error1"))
//			return nil
//		})
//		_ = gs.Go(func() error {
//			panic("panic5")
//			return nil
//		})
//		_ = gs.Go(func() error {
//			gs.Err(errors.New("record"))
//			return nil
//		})
//		err := gs.Wait()
//		fmt.Println(err)
//	}
//	time.Sleep(MaxIdleWorkerDuration)
//}

func BenchmarkReuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gs := New(Config{
			GoCount: 1000,
			Wait:    true,
		})
		for j := 0; j < 1000; j++ {
			_ = gs.Go(func() error {
				return nil
			})
		}

		_ = gs.Wait()
	}
}

func BenchmarkNotReuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gs := New(Config{
			GoCount:  1000,
			Wait:     true,
			NotReuse: true,
		})
		for j := 0; j < 1000; j++ {
			_ = gs.Go(func() error {
				return nil
			})
		}
		_ = gs.Wait()
	}
}
