package pool

import "sync/atomic"

// Pool describes goroutine pool for reducing a number of goroutine allocations
type Pool struct {
	WCh  chan func()
	Size int32
	Used *int32
}

// New creates new goroutine pool
func New(n int) *Pool {
	var q int32
	used := &q
	wCh := make(chan func())
	for i := 0; i < n; i += 1 {
		go Worker(wCh, used)
	}
	return &Pool{
		WCh:  wCh,
		Size: int32(n),
		Used:used,
	}
}

// Run sends new job to goroutine pool
func (p *Pool) Run(f func()) {
	atomic.AddInt32(p.Used, 1)
	p.WCh <- f
	if *p.Used == p.Size {
		p.ExpandTwice()
	}
}

func (p *Pool) ExpandTwice() {
	a := p.Size
	atomic.AddInt32(&p.Size, p.Size)
	go func() {
		for i := a; i >= 0; i-- {
			go Worker(p.WCh, p.Used)
		}
	}()
}

// Worker works :)
func Worker(job chan func(), used *int32) {
	for {
		f := <-job
		f()
		atomic.AddInt32(used, -1)
	}
}
