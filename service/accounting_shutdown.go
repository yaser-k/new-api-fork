package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

// pendingAccounting counts billing writes that still run after their request
// handler returned, so shutdown can wait for them before the final flush.
var pendingAccounting struct {
	mu    sync.Mutex
	count int
	idle  chan struct{} // closed when count drops to zero
}

// goAccounting runs fn on the goroutine pool as tracked accounting work.
func goAccounting(fn func()) {
	pendingAccounting.mu.Lock()
	if pendingAccounting.count == 0 {
		pendingAccounting.idle = make(chan struct{})
	}
	pendingAccounting.count++
	pendingAccounting.mu.Unlock()

	gopool.Go(func() {
		defer func() {
			pendingAccounting.mu.Lock()
			pendingAccounting.count--
			if pendingAccounting.count == 0 {
				close(pendingAccounting.idle)
			}
			pendingAccounting.mu.Unlock()
		}()
		fn()
	})
}

// ShutdownAccounting writes the accounting a stopping process has already
// accepted. Call it after the HTTP server has shut down: it waits up to
// waitTimeout for tracked asynchronous billing work, then stops the batch
// updater and flushes every pending batch delta. Requests still running when
// the server shutdown timed out are not covered.
func ShutdownAccounting(waitTimeout time.Duration) error {
	var errs []error

	start := time.Now()
	pendingAccounting.mu.Lock()
	waiting, idle := pendingAccounting.count, pendingAccounting.idle
	pendingAccounting.mu.Unlock()
	if waiting == 0 {
		common.SysLog("shutdown: no async billing work pending")
	} else {
		timer := time.NewTimer(waitTimeout)
		select {
		case <-idle:
			common.SysLog(fmt.Sprintf("shutdown: waited for %d async billing jobs in %s", waiting, time.Since(start)))
		case <-timer.C:
			pendingAccounting.mu.Lock()
			left := pendingAccounting.count
			pendingAccounting.mu.Unlock()
			err := fmt.Errorf("gave up waiting for async billing jobs after %s: %d still running, their writes after the final flush are lost", waitTimeout, left)
			common.SysError("shutdown: " + err.Error())
			errs = append(errs, err)
		}
		timer.Stop()
	}

	if !common.BatchUpdateEnabled {
		return errors.Join(errs...)
	}

	start = time.Now()
	model.StopBatchUpdater()
	common.SysLog(fmt.Sprintf("shutdown: batch updater stopped in %s", time.Since(start)))

	start = time.Now()
	result := model.FlushBatchUpdates()
	common.SysLog(fmt.Sprintf("shutdown: flushed batch updates in %s: users=%d tokens=%d channels=%d failed=%d",
		time.Since(start), result.Users, result.Tokens, result.Channels, result.Failed))
	if result.Failed > 0 {
		err := fmt.Errorf("%d batch update writes failed during the final flush", result.Failed)
		common.SysError("shutdown: " + err.Error())
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
