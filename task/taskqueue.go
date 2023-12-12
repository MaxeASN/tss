package task

import (
	"errors"
	"log"
	"sync"
)

var ErrQueueStopped = errors.New("QueueStopped")
var ErrTimeout = errors.New("Timeout")

type TaskQueue interface {
	Next() (any, error)
	Add(item any) error
	Close()
	Stopped() bool
	Len() int
}

type Queue struct {
	queue   []any
	cond    *sync.Cond
	stopped bool
	len     int
}

func NewTaskQueue() *Queue {
	return &Queue{
		queue:   make([]any, 0),
		cond:    sync.NewCond(&sync.Mutex{}),
		stopped: false,
		len:     0,
	}
}

func (q *Queue) Next() (any, error) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	// if stopped return immediately
	if q.stopped {
		return nil, ErrQueueStopped
	}

	// check if the queue is not stopped and empty
	for q.len == 0 {
		log.Print("Queue is empty, waiting...")
		q.cond.Wait()
		log.Print("Queue is likely not empty, quitting...")
	}
	log.Print("Queue is not empty, got task...")
	// then return the first item in the queue
	item := q.queue[0]
	q.queue = q.queue[1:]
	// renew queue length
	q.len--
	return item, nil
}

func (q *Queue) Add(item any) error {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	// if stopped, return immediately
	if q.stopped {
		return ErrQueueStopped
	}
	// append the item to the queue
	q.queue = append(q.queue, item)
	// cal the length of the queue
	q.len = len(q.queue)
	// awake the sleep goroutine
	q.cond.Signal()
	return nil
}

func (q *Queue) Close() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	// if already stopped, return immediately
	if q.stopped {
		return
	}
	// mark the queue as stopped
	q.stopped = true
	// wake up all waiting goroutines
	q.cond.Broadcast()
	return
}

func (q *Queue) Stopped() bool {
	// return the stopped state
	return q.stopped
}

func (q *Queue) Len() int {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	// return the number of items in the queue
	return q.len
}
