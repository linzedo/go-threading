package g

import (
	"errors"
	"fmt"
	"go.uber.org/atomic"
	"os"
	"runtime"
	"strings"
	"sync"
)

var path string

func init() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	path = strings.ReplaceAll(dir, `\`, "/")
	SetColor(true)
}

type (
	Config struct {
		Limit   int  //限制同时存在的协程数，0则不受限制
		GoCount int  //要控制的协程数量
		Wait    bool //是否等待所有goroutine执行完毕再关闭,遇到错误可立即结束阻塞,默认不等
	}

	GoSync struct {
		panic *errorsSafe
		wChan *chanm[struct{}]

		workers int64        //待开启的
		working atomic.Int64 //正在执行的
		done    atomic.Int64 //已结束的

		limit *chanm[struct{}]
		wait  bool //是否等待协程组结束后结束阻塞

		errs        *errorsSafe
		errChan     *chanm[error]
		errChanOnce sync.Once
		finish      bool //已经结束的
		mu          sync.Mutex
	}

	chanm[T any] struct {
		ch   chan T
		once sync.Once
	}
)

func newChanm[T any](buf int) *chanm[T] {
	c := chanm[T]{
		ch: make(chan T, buf),
	}
	return &c
}

func (c *chanm[T]) getChan() chan T {
	return c.ch
}

func (c *chanm[T]) close() {
	c.once.Do(func() {
		if c != nil {
			close(c.ch)
		}
	})
}

func New(config Config) *GoSync {
	g := newGoS(config.GoCount)
	if config.Limit > 0 {
		g.limit = &chanm[struct{}]{ch: make(chan struct{}, config.Limit)}
	}
	g.wait = config.Wait
	return g
}

func newGoS(goCount int) *GoSync {
	g := &GoSync{}
	g.workers = int64(goCount)
	g.working.Store(0)
	g.done.Store(0)
	g.wChan = newChanm[struct{}](0)
	g.errChan = newChanm[error](1)
	g.errs = newError()
	g.panic = newError()
	return g
}

func (g *GoSync) Go(f func() error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.finish {
		return errors.New("goroutine group control has ended")
	}

	if g.working.Add(1) > g.workers {
		return errors.New("the number of goroutines created exceeds the limit")
	}

	if g.limit != nil && !g.finish {
		g.limit.getChan() <- struct{}{}
	}

	g.goSafe(f)

	return nil
}

func (g *GoSync) goSafe(f func() error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("%s,%s", getPanicCtx(), Red.Add(fmt.Sprintf("%v", r)))
				g.panic.append(err)
				if !g.wait {
					g.mu.Lock()
					g.finish = true
					g.mu.Unlock()
					g.wChan.close()
				}
			}
			if g.done.Add(1) == g.workers {
				g.wChan.close()
			}
			if g.limit != nil {
				<-g.limit.getChan()
			}
		}()
		g.Err(f())
	}()
}

// Err 在异步任务中主动收集错误
// if err!=nil{
// Err(err)
// }...
// Or
//
//	if err!=nil{
//	 return err
//	}
func (g *GoSync) Err(err error) {
	if err == nil {
		return
	}

	if g.wait {
		g.errs.append(err)
		return
	}

	g.errChanOnce.Do(func() {
		g.mu.Lock()
		g.finish = true
		g.mu.Unlock()
		g.errChan.getChan() <- err
	})
}

// Wait wait=false(默认):如何执行的异步任务出现了err或者panic会立刻返回，未启动的任务也不再启动,返回的err信息将会是1个err或者一个panicErr
// wait=true:会等待所有任务执行完毕才会返回,会返回全部的panicErr和err
func (g *GoSync) Wait() error {
	defer g.close()
	select {
	case <-g.wChan.getChan():
	case err := <-g.errChan.getChan():
		return err
	}

	panicErr := g.panic.errs
	if len(panicErr) > 0 && !g.wait {
		return panicErr[0]
	}
	var b strings.Builder
	msg := ""
	if len(panicErr) > 0 {
		msg = panicErr[0].Error()
		for i := 1; i < len(panicErr); i++ {
			msg = fmt.Sprintf("%s;%s", msg, panicErr[i])
		}
		b.WriteString(Red.Add("PANIC:"))
		b.WriteString(msg)
	}

	es := g.errs
	if len(es.errs) > 0 {
		msg = es.Error()
		b.Grow(len(msg) + 10)
		b.WriteString("\n")
		b.WriteString(Red.Add("ERROR:"))
		b.WriteString(msg)
	}
	if b.Len() > 0 {
		return errors.New(b.String())
	}
	return nil
}

func (g *GoSync) close() {
	g.mu.Lock()
	g.finish = true
	g.mu.Unlock()

	g.errChan.close()
	//g.limit.close()
	g.wChan.close()

}

// getPanicCtx 获取 panic 发生的位置
func getPanicCtx() string {
	var name, file string
	var line int
	var pc = make([]uintptr, 16)

	_ = runtime.Callers(3, pc[:])
	frames := runtime.CallersFrames(pc)
	var b strings.Builder
	var color Color
	add := func(s string) string {
		if color == Blue {
			color = Cyan
		} else {
			color = Blue
		}
		return color.Add(s)
	}
	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		file = frame.File
		line = frame.Line
		name = frame.Function
		if path == "" && !strings.HasPrefix(name, "runtime.") {
			_, _ = b.WriteString(add(fmt.Sprintf("(%s:%d)", name, line)))
			break
		} else if path != "" && strings.HasPrefix(file, path) {
			_, _ = b.WriteString(add(fmt.Sprintf("(%s:%d)", name, line)))
			break
		} else if path != "" && !strings.HasPrefix(name, "runtime.") {
			_, _ = b.WriteString(add(fmt.Sprintf("(%s:%d)", name, line)))
		}

	}
	if b.Len() > 0 {
		return b.String()
	}

	return fmt.Sprintf("pc:%x", pc)
}
