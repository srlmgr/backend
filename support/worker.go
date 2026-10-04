package support

import (
	"sync"

	"github.com/samber/lo"

	"github.com/srlmgr/backend/log"
)

type (
	resultData[O any] struct {
		Result O
		err    error
	}

	job[I any, O any] struct {
		Args     I
		ResultCh chan<- resultData[O]
	}
	WorkerOption[O any] func(*workerCfg[O])

	// idx is the index of the incoming request list in Worker.Process()
	ResultCallback[O any] func(idx int, res O, err error)
	Task[I, O any]        func(args I) (res O, err error)
	workerCfg[O any]      struct {
		numWorker      int
		resultCallback ResultCallback[O]
		logger         *log.Logger
	}
	Worker[I, O any] struct {
		cfg  *workerCfg[O]
		task Task[I, O]
	}
)

func defaultWorkerConfig[O any]() *workerCfg[O] {
	return &workerCfg[O]{
		numWorker: 1,
		logger:    log.Default().Named("support.worker"),
	}
}

func NewWorker[I, O any](t Task[I, O], opts ...WorkerOption[O]) *Worker[I, O] {
	cfg := defaultWorkerConfig[O]()
	for _, o := range opts {
		o(cfg)
	}
	ret := &Worker[I, O]{
		cfg:  cfg,
		task: t,
	}
	return ret
}

func WithResultCallback[O any](arg ResultCallback[O]) WorkerOption[O] {
	return func(c *workerCfg[O]) {
		c.resultCallback = arg
	}
}

func WithNumWorker[O any](arg int) WorkerOption[O] {
	return func(c *workerCfg[O]) {
		c.numWorker = arg
	}
}

func WithLogger[O any](arg *log.Logger) WorkerOption[O] {
	return func(c *workerCfg[O]) {
		c.logger = arg
	}
}

func (w *Worker[I, O]) Process(input []I) {
	mainInputChan := make(chan job[I, O])
	inpChan := lo.FanIn(2, mainInputChan)

	go func() {
		var wg sync.WaitGroup

		ret := make([]I, len(input))
		_ = ret
		wg.Add(len(input))
		for i := range input {
			resultCh := make(chan resultData[O])
			mainInputChan <- job[I, O]{
				Args:     input[i],
				ResultCh: resultCh,
			}
			go func(idx int, resCh chan resultData[O]) {
				res := <-resCh

				if w.cfg.resultCallback != nil {
					w.cfg.resultCallback(idx, res.Result, res.err)
				}

				wg.Done()
			}(i, resultCh)
		}
		wg.Wait()
		close(mainInputChan)
	}()
	var wgWorker sync.WaitGroup
	wgWorker.Add(w.cfg.numWorker)
	for i := range w.cfg.numWorker {
		go func(id int) {
			defer wgWorker.Done()
			w.cfg.logger.Debug("worker started", log.Int("id", id))
			for job := range inpChan {
				result, err := w.task(job.Args)
				job.ResultCh <- resultData[O]{Result: result, err: err}
			}
			w.cfg.logger.Debug("worker finished", log.Int("id", id))
		}(i)
	}
	wgWorker.Wait()
	w.cfg.logger.Debug("all workers finished")
}
