package threading

import (
	"math"
	"runtime"
	"sync"
	"time"
)

const (
	DefaultMaxWorkCount   = 256 * 1024
	DefaultOverJobCount   = 10
	MaxIdleWorkerDuration = time.Second * 20
)

type (
	goPool struct {
		MaxWorkCount          int
		MinWorkCount          int
		MaxIdleWorkerDuration time.Duration

		going      []*goChan //就绪队列，可获取已就绪routine的通讯,写入/移除过程保证有序
		goCount    int       //正在执行任务数量
		goChanPool sync.Pool //通讯池
		gsPool     sync.Pool //对象池
		mu         sync.Mutex
	}

	goChan struct {
		lastUseTime time.Time
		task        chan *task
	}

	PoolOpt func(*goPool)

	task struct {
		job func() error
		gs  *GoSync
	}
)

var (
	gp     *goPool
	allJob chan *task
	once   sync.Once
)

func SetMaxWorkCount(count int) PoolOpt {
	return func(*goPool) {
		if gp == nil || count == 0 {
			return
		}
		if count < 0 {
			gp.MaxWorkCount = math.MaxInt
		}
		gp.MaxWorkCount = count
		if gp.MinWorkCount > gp.MaxWorkCount && gp.MaxWorkCount > 0 {
			gp.MinWorkCount = gp.MaxWorkCount
		}
	}
}

func SetMinWorkCount(count int) PoolOpt {
	return func(*goPool) {
		if gp == nil || count == 0 {
			return
		}
		if count > 0 {
			gp.MinWorkCount = count
		}
		if gp.MinWorkCount > gp.MaxWorkCount && gp.MaxWorkCount > 0 {
			gp.MinWorkCount = gp.MaxWorkCount
		}
	}
}

func SetMaxIdleWorkerDuration(duration time.Duration) PoolOpt {
	return func(p *goPool) {
		if gp == nil || duration <= 0 {
			return
		}
		gp.MaxIdleWorkerDuration = duration
	}
}

func tryGetTask() *task {
	select {
	case t := <-allJob:
		return t
	default:
		return nil
	}
}

var goChanCap = func() int {
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}
	return 1
}()

func newGoPool() *goPool {
	g := &goPool{
		MaxIdleWorkerDuration: MaxIdleWorkerDuration,
		MaxWorkCount:          DefaultMaxWorkCount,
	}
	g.goChanPool.New = func() any {
		return &goChan{task: make(chan *task, goChanCap)}
	}
	g.gsPool.New = func() any {
		return newGoS()
	}
	return g
}

func newWorkerChan() *goChan {
	return &goChan{
		task: make(chan *task, 0),
	}
}

func runGlobalTasks(t *task) {
	gs := t.gs
	defer gs.recover()
	gs.Err(t.job())
}

func StartPool(opt ...PoolOpt) {
	once.Do(func() {
		initPath()
		gp = newGoPool()
		for i := range opt {
			opt[i](gp)
		}
		gp.start()
	})
}

func (p *goPool) start() {
	allJob = make(chan *task, DefaultOverJobCount)
	p.mu.Lock()
	need := p.MinWorkCount - len(p.going)
	if need > 0 {
		p.going = make([]*goChan, 0, need)
		for i := 0; i < need; i++ {
			ch := p.goChanPool.Get().(*goChan)
			go func() {
				//执行任务
				p.goFunc(ch)
			}()
			p.going = append(p.going, ch)
		}
	}
	p.mu.Unlock()
	//开启清扫任务
	go func() {
		var reuse []*goChan
		for {
			p.clean(&reuse)
			select {
			default:
				time.Sleep(p.MaxIdleWorkerDuration)
			}
		}
	}()
	//go开启全局任务处理
	go func() {
		for t := range allJob {
			runGlobalTasks(t)
		}
	}()
}

//获取资源并传递任务
func (p *goPool) Serve(j *task) {
	ch := p.getCh()
	if ch == nil {
		allJob <- j
		return
	}
	ch.task <- j
	return
}

//获取资源
func (p *goPool) getCh() *goChan {
	var ch *goChan
	newGo := false

	//尝试获取与正在闲置的goroutine的通信或者获取创建新routine的权限
	p.mu.Lock()
	going := p.going
	last := len(going) - 1
	if last < 0 {
		if p.goCount < p.MaxWorkCount {
			newGo = true
			p.goCount++
		}
	} else {
		ch = going[last]
		going[last] = nil
		p.going = going[:last]
	}

	p.mu.Unlock()
	if ch == nil {
		if !newGo {
			return nil
		}
		ch = p.goChanPool.Get().(*goChan)
		go func() {
			//执行任务
			p.goFunc(ch)
			p.goChanPool.Put(ch)
		}()
	}
	return ch
}

func (p *goPool) Going() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.goCount
}

func (p *goPool) goFunc(ch *goChan) {
	var (
		t *task
	)
	defer func() {
		p.mu.Lock()
		p.goCount--
		p.mu.Unlock()
	}()

	for t = range ch.task {
		//用于清扫协程进行关闭通知
		if t == nil {
			break
		}
	help:
		//	执行内容
		runTask(t)

		//尝试从全局中获取数据消费
		if t = tryGetTask(); t != nil {
			goto help
		}

		//记录最后一次通讯信息
		p.release(ch)
		t = nil
	}
}

func runTask(t *task) {
	gs := t.gs
	defer gs.recover()
	gs.Err(t.job())
}

func (p *goPool) release(ch *goChan) {
	ch.lastUseTime = time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.going = append(p.going, ch)
}

//最大闲置清扫
func (p *goPool) clean(reuse *[]*goChan) {
	maxIdleGoDuration := p.MaxIdleWorkerDuration
	currentTime := time.Now().Add(-maxIdleGoDuration)

	p.mu.Lock()
	going := p.going
	park := len(going)
	//清除超过闲置时间未使用的通讯
	l, r, mid := 0, park-1, 0
	for l <= r {
		mid = (l + r) / 2
		if currentTime.After(p.going[mid].lastUseTime) {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	i := r - p.MinWorkCount
	if i < 0 {
		p.mu.Unlock()
		return
	}

	*reuse = append((*reuse)[:0], going[:i+1]...)
	m := copy(going, going[i+1:])
	for i = m; i < park; i++ {
		going[i] = nil
	}
	p.going = going[:m]
	p.mu.Unlock()

	tmp := *reuse
	for i := range tmp {
		tmp[i].task <- nil
		tmp[i] = nil
	}
}
