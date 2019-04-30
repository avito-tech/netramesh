package pool

// Pool describes goroutine pool for reducing a number of goroutine allocations
type Pool struct {
	WCh  chan func()
	Size int
}

// New creates new goroutine pool
func New(n int) *Pool {
	wCh := make(chan func(), 100)
	for i := 0; i < n; i += 1 {
		go Worker(wCh)
	}
	return &Pool{
		WCh:  wCh,
		Size: n,
	}
}

// Run sends new job to goroutine pool
func (p *Pool) Run(f func()) {
	p.WCh <- f
}

// Worker works :)
func Worker(job chan func()) {
	for {
		f := <-job
		f()
	}
}
