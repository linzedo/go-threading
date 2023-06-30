package g

import (
	"errors"
	"fmt"
	"go.uber.org/goleak"
	"testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
	m.Run()
}

func TestGoWait(t *testing.T) {
	gs := New(Config{
		Limit:   5,
		GoCount: 4,
		Wait:    true,
	})
	_ = gs.Go(func() error {
		gs.Err(errors.New("error1"))
		return nil
	})
	_ = gs.Go(func() error {
		return errors.New("error2")
	})
	_ = gs.Go(func() error {
		panic("panic3")
		return nil
	})
	_ = gs.Go(func() error {
		panic("panic4")
		return nil
	})
	fmt.Println("over:", gs.Go(func() error {
		panic("panic5")
		return nil
	}))

	err := gs.Wait()
	fmt.Println(err)

}

func TestWithoutWait(t *testing.T) {
	for i := 0; i < 100; i++ {
		gs := New(Config{
			Limit:   5,
			GoCount: 4,
			Wait:    false,
		})
		_ = gs.Go(func() error {
			panic("panic4")
			return nil
		})
		_ = gs.Go(func() error {
			gs.Err(errors.New("error1"))
			return nil
		})
		_ = gs.Go(func() error {
			panic("panic5")
			return nil
		})
		_ = gs.Go(func() error {
			gs.Err(errors.New("record"))
			return nil
		})
		err := gs.Wait()
		fmt.Println(err)
	}
}
