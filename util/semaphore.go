package util

type Semaphore chan bool

func NewSemaphore(threads int) Semaphore {
	sem := Semaphore(make(chan bool, threads))
	return sem
}

func (s Semaphore) Acquire() {
	s <- true
}

func (s Semaphore) Release() {
	<-s
}
