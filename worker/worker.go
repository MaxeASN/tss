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

	// waitgroup to wait for all workers to finish
	wg sync.WaitGroup

	// quit is used to stop the worker
	quit chan struct{}

	// queue is used to sync tasks to the worker
	queue TaskQueue

	// workers is used to reuse the subworker
	workers chan *SubWorker

	// subworker is the worker index of the task
	subWorkerIndex map[string]*SubWorker

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
		config:         cfg,
		quit:           make(chan struct{}),
		queue:          NewTaskQueue(),
		workers:        workers,
		subWorkerIndex: make(map[string]*SubWorker),
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
				next, err := w.queue.Next()
				if err != nil {
					log.Info("get task error", "error", err)
					continue
				}
				w.wg.Add(1)
				go func() {
					// decode task
					task := next.(*TaskEvent)
					// try to get subworker
					//subWorker := <-w.workers
					subWorker := w.subWorkerIndex[task.Tid]
					// try to process the task
					subWorker.process(task)
					// return the subworker to the worker pool
					w.workers <- subWorker
					delete(w.subWorkerIndex, task.Tid)
					w.wg.Done()
				}()
			}
		}
	}()
}

func (w *Worker) PushEvent(event *TaskEvent) (chan TaskResult, error) {

	subWorker := <-w.workers
	w.subWorkerIndex[event.Tid] = subWorker
	log.Info("Get subworker", "name", subWorker.name)

	if err := w.queue.Add(event); err != nil {
		return nil, err
	}
	return subWorker.ch, nil
}

func (w *Worker) ReleaseWorker(tid string) {
	subworker := w.subWorkerIndex[tid]
	w.workers <- subworker
	delete(w.subWorkerIndex, tid)
	log.Info("!!! Release subworker", "name", subworker.name, "task_id", tid)
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

	// wait for the subworkers to finish
	w.wg.Wait()

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
		sw.ch <- TaskResult{
			Err:    nil,
			Tid:    task.Tid,
			Result: "ok",
		}
	}
}
