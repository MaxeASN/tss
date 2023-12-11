package task

import (
	"context"
	"errors"
	"github.com/bnb-chain/tss/common"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var handlerError = errors.New("error handling message")

type Worker struct {
	ctx context.Context

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

	// prepare handler
	prepareHandler PrepareHandler

	// bootstrapper
	isBootstrapper bool

	// workerLen is the number of subworkers
	// workerLen int
}

type SubWorker struct {
	// ctx when the request is cancelled, subworker can be cancelled immediately
	ctx context.Context

	// name is the name of the subworker
	name string

	// task
	task *TaskEvent

	// handler
	handler IEventHandler

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
	Op common.OpType
}

type PrepareHandler func(ctx context.Context, event *TaskEvent) (result bool)
type HandlerFactory func(ctx context.Context) (handler IEventHandler)

func NewWorker(ctx context.Context, isBootstrapper bool, cfg *WorkerConfig, ph PrepareHandler, hf HandlerFactory) *Worker {
	workers := make(chan *SubWorker, cfg.WorkerLimit)
	// generate subworkers
	for i := 0; i < cfg.WorkerLimit; i++ {
		h := hf(ctx)
		workers <- &SubWorker{
			ctx:     ctx,
			name:    "subworker-" + strconv.Itoa(i),
			ch:      make(chan TaskResult, 0),
			stopped: false,
			handler: h,
		}
	}
	log.Info("Generated subworkers", "len", cfg.WorkerLimit)
	// new Worker
	return &Worker{
		ctx:            ctx,
		config:         cfg,
		quit:           make(chan struct{}),
		queue:          NewTaskQueue(),
		workers:        workers,
		subWorkerIndex: make(map[string]*SubWorker),
		isBootstrapper: isBootstrapper,
	}
}

func (w *Worker) Start() {
	go func() {
		timer := time.NewTicker(time.Second * 1)
		for {

			select {
			case <-w.quit:
				log.Info("Worker stopping...")
				return
			case <-timer.C:
				timer.Reset(time.Second * 5)
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
					// prepare localparty
					if w.isBootstrapper {
						if ok := w.prepareHandler(w.ctx, task); !ok {
							w.CancelTask(task.Tid)
							w.wg.Done()
							return
						}
					}

					// try to get subworker
					w.lock.RLock()
					subWorker := w.subWorkerIndex[task.Tid]
					w.lock.RUnlock()

					// preset the event task
					subWorker.preset(task)

					// try to process the task
					subWorker.process()

					w.releaseWorkerIndex(task.Tid)

					w.releaseWorker(subWorker)

					w.wg.Done()
				}()
			}
		}
	}()
}

func (w *Worker) CancelTask(tid string) {
	w.lock.RLock()
	subworker := w.subWorkerIndex[tid]
	w.lock.RUnlock()

	// release the subworker index
	w.releaseWorkerIndex(tid)
	// return the subworker to the worker pool
	w.releaseWorker(subworker)
	log.Info("!!! Release subworker", "name", subworker.name, "task_id", tid)
}

func (w *Worker) PushEvent(event *TaskEvent) (chan TaskResult, error) {
	if _, ok := w.subWorkerIndex[event.Tid]; ok {
		return nil, errors.New("task already exists")
	}
	subWorker := <-w.workers
	w.subWorkerIndex[event.Tid] = subWorker
	log.Info("Get subworker", "name", subWorker.name, "Tid", event.Tid)

	if err := w.queue.Add(event); err != nil {
		return nil, err
	}
	return subWorker.ch, nil
}

func (w *Worker) releaseWorkerIndex(tid string) {
	w.lock.Lock()
	delete(w.subWorkerIndex, tid)
	w.lock.Unlock()
}

func (w *Worker) releaseWorker(subworker *SubWorker) {
	w.workers <- subworker
}

func (w *Worker) GenEventKey(event *TaskEvent) error {
	message := serialize(event)
	if message == nil {
		return errors.New("message invalid")
	}

	event.Tid = key
	return nil
}

func serialize(event *TaskEvent) []byte {
	if len(event.Message) != 66 {
		return nil
	}
	buf := make([]byte, 0)
	buf = append(buf, []byte(strconv.Itoa(event.ChainId))...)

	addrBytes, _ := hexutil.Decode(event.Address)
	msgBytes, _ := hexutil.Decode(event.Message)
	buf = append(buf, addrBytes...)
	buf = append(buf, msgBytes...)

	return buf
}

func (w *Worker) Stop() {

	//w.lock.Lock()
	//defer w.lock.Unlock()

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

//func (w *Worker) GetChannel() chan {}

// preset init the subworker with task msg
func (sw *SubWorker) preset(event *TaskEvent) {
	sw.task = event
	sw.handler.Preprocess(event)
}

type IEventHandler interface {
	Preprocess(event *TaskEvent) bool
	Start()
	Result() <-chan any
	Error() <-chan error
}

// process processes the task
func (sw *SubWorker) process() {
	go func() { sw.handler.Start() }()
	select {
	case <-sw.ctx.Done():
		log.Info("Processing Finished as unusual", "id", sw.task.Tid, "error", sw.ctx.Err())
		sw.ch <- TaskResult{
			Tid:    sw.task.Tid,
			Result: "error",
			Err:    errors.New("processing cancelled"),
		}
	case err := <-sw.handler.Error(): // error handling
		sw.ch <- TaskResult{
			Tid:    sw.task.Tid,
			Result: "error",
			Err:    err,
		}
	case result := <-sw.handler.Result(): // result handling
		sw.ch <- TaskResult{
			Tid:    sw.task.Tid,
			Result: result,
			Err:    nil,
		}
	}
}
