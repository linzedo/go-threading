package threading

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

func initPath() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	path = strings.ReplaceAll(dir, `\`, "/")
	SetColor(true)
}

type (
	Config struct {
		Limit    int  //限制同时存在的协程数，0则不受限制
		GoCount  int  //要控制的协程数量
		Wait     bool //是否等待所有goroutine执行完毕再关闭,遇到错误可立即结束阻塞,默认不等
		NotReuse bool //是否开启协程复用模式，默认开启
	}

	GoSync struct {
		panic errorsSafe
		wChan chanOnce[struct{}]

		workers int64        //待开启的
		working atomic.Int64 //正在执行的
		done    atomic.Int64 //已结束的

		limit chan struct{}
		wait  bool //是否等待协程组结束后结束阻塞
		reuse bool

		errs        errorsSafe
		errChan     chan error
		errChanOnce sync.Once
		finish      bool //已经结束的
		mu          sync.Mutex
	}

	chanOnce[T any] struct {
		ch     chan T
		mu     sync.Mutex
		send   int
		closed bool
	}
)

func newChanOnce[T any](buf int) *chanOnce[T] {
	c := chanOnce[T]{
		ch: make(chan T, buf),
	}
	return &c
}

func (c *chanOnce[T]) getChan() chan T {
	return c.ch
}

func (c *chanOnce[T]) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
	}
}

func (c *chanOnce[T]) sentOnce(msg T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.send == 0 {
		c.send++
		c.ch <- msg
	}
}

func New(config Config) *GoSync {
	StartPool()

	var g *GoSync
	if gp != nil {
		g = gp.gsPool.Get().(*GoSync)
	}

	g.reset(config)

	return g
}

func newGoS() *GoSync {
	g := &GoSync{
		wChan: chanOnce[struct{}]{
			ch: make(chan struct{}, 1),
		},
		errChan: make(chan error, 1),
		//errs:    newError(),
		//panic:   newError(),
	}

	return g
}

func (g *GoSync) reset(config Config) {
	g.workers = int64(config.GoCount)
	g.wait = config.Wait
	g.reuse = !config.NotReuse
	g.errChanOnce = sync.Once{}
	g.working.Store(0)
	g.done.Store(0)
	if config.Limit > 0 && cap(g.limit) != config.Limit {
		g.limit = make(chan struct{}, config.Limit)
	} else if config.Limit == 0 {
		g.limit = nil
	}

	if g.wChan.ch != nil {
		g.wChan.closed = false
		g.wChan.send = 0
	}
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
		g.limit <- struct{}{}
	}
	if g.reuse {
		gp.Serve(&task{
			job: f,
			gs:  g,
		})
		return nil
	}

	g.goSafe(f)

	return nil
}

func (g *GoSync) goSafe(f func() error) {
	go func() {
		defer g.recover()

		g.Err(f())
	}()
}

func (g *GoSync) recover() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%s,%s", getPanicCtx(), Red.Add(fmt.Sprintf("%v", r)))
		g.panic.append(err)
		if !g.wait {
			g.mu.Lock()
			g.finish = true
			g.mu.Unlock()
			g.wChan.sentOnce(struct{}{})
		}
	}
	if g.done.Add(1) == g.workers {
		g.wChan.sentOnce(struct{}{})
	}
	if g.limit != nil {
		<-g.limit
	}
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
		g.errChan <- err
	})
}

// Wait wait=false(默认):如何执行的异步任务出现了err或者panic会立刻返回，未启动的任务也不再启动,返回的err信息将会是1个err或者一个panicErr
// wait=true:会等待所有任务执行完毕才会返回,会返回全部的panicErr和err
func (g *GoSync) Wait() error {
	defer func() {
		g.panic.errs = nil
		g.errs.errs = nil
		gp.gsPool.Put(g)
	}()
	select {
	case <-g.wChan.getChan():
	case err := <-g.errChan:
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

	es := g.errs.errs
	if len(es) > 0 {
		msg = g.errs.Error()
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

	close(g.errChan)
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
