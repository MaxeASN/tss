package worker

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type Worker struct {
	config *WorkerConfig
	// lock protects access
	lock sync.RWMutex

	// quit is used to stop the worker
	quit chan struct{}

	// queue is used to sync tasks to the worker
	queue TaskQueue

	// workers is used to reuse the subworker
	workers chan *SubWorker

	// workerLen is the number of subworkers
	// workerLen int
}

type SubWorker struct {
	// ctx when the request is cancelled, subworker can be cancelled immediately
	ctx context.Context

	// name is the name of the subworker
	name string

	// ch is used to notify the request is done
	ch chan TaskResult

	// stopped is the status of the subworker
	stopped bool
}

type TaskResult struct {
	// error message of the task
	Err error

	// task id
	Tid string

	// task result
	Result any
}

type TaskEvent struct {
	// task id
	Tid string

	// event chainid
	ChainId int

	// event address
	Address string

	// event data
	Message string

	// event type
	Op OpType
}

type OpType int

const (
	Init OpType = iota
	GenChannelId
	GenKey
	Sign
	Regroup
	Verify
)

func NewWorker(ctx context.Context, cfg *WorkerConfig) *Worker {
	workers := make(chan *SubWorker, cfg.WorkerLimit)
	// generate subworkers
	for i := 0; i < cfg.WorkerLimit; i++ {
		workers <- &SubWorker{
			ctx:     ctx,
			name:    "subworker-" + strconv.Itoa(i),
			ch:      make(chan TaskResult, 0),
			stopped: false,
		}
	}
	log.Info("Starting subworkers", "len", cfg.WorkerLimit)
	// new Worker
	return &Worker{
		config:  cfg,
		quit:    make(chan struct{}),
		queue:   NewTaskQueue(),
		workers: workers,
	}
}

func (w *Worker) Start() {
	go func() {
		for {

			select {
			case <-w.quit:
				log.Info("Worker stopping...")
				return
			//case <-w.queue.Next():
			case <-time.After(time.Second * 1):
				fmt.Println("Goroutine number: ", runtime.NumGoroutine())
			default:
				if next, err := w.queue.Next(); err != nil {
					log.Info("get task error", "error", err)
					continue
				} else {
					// decode task
					task := next.(*TaskEvent)
					// try to get subworker
					subWorker := <-w.workers
					// try to process the task
					subWorker.process(task)
					// return the subworker to the worker pool
					w.workers <- subWorker
				}
			}
		}
	}()
}

func (w *Worker) PushEvent(event *TaskEvent) error {
	return w.queue.Add(event)
}

func (w *Worker) Stop() {

	w.lock.Lock()
	defer w.lock.Unlock()

	// stop all the workers
	go func() {
		for sw := range w.workers {
			sw.stopped = true
			// close the channel
			close(sw.ch)
			log.Info("Subworker Stopped", "name", sw.name)
		}
	}()

	// close the queue
	log.Info("Waiting for queue to exit...")
	w.queue.Close()

	// stop the worker itself
	close(w.quit)
}

func (sw *SubWorker) process(task *TaskEvent) {
	select {
	case <-sw.ctx.Done():
		log.Info("Processing Paused", "id", task.Tid)
		return
	default:
		log.Info("Processing task", "id", task.Tid)
		time.Sleep(time.Second * 2)
	}
}
