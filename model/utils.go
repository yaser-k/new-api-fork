package model

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

const (
	BatchUpdateTypeUserQuota = iota
	BatchUpdateTypeTokenQuota
	BatchUpdateTypeUsedQuota
	BatchUpdateTypeChannelUsedQuota
	BatchUpdateTypeRequestCount
	BatchUpdateTypeCount // if you add a new type, you need to add a new map and a new lock
)

var batchUpdateStores []map[int]int
var batchUpdateLocks []sync.Mutex

func init() {
	for range BatchUpdateTypeCount {
		batchUpdateStores = append(batchUpdateStores, make(map[int]int))
		batchUpdateLocks = append(batchUpdateLocks, sync.Mutex{})
	}
}

// batchFlushLock serializes flushes, so a flush returns only after every
// delta taken out of the stores before it started has been written.
var batchFlushLock sync.Mutex

var batchUpdaterLock sync.Mutex
var batchUpdaterCancel context.CancelFunc
var batchUpdaterDone chan struct{}

func InitBatchUpdater() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	batchUpdaterLock.Lock()
	batchUpdaterCancel = cancel
	batchUpdaterDone = done
	batchUpdaterLock.Unlock()
	gopool.Go(func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(common.BatchUpdateInterval) * time.Second):
				FlushBatchUpdates()
			}
		}
	})
}

// StopBatchUpdater stops the periodic batch writer and waits for a flush it
// is running, so that a final FlushBatchUpdates is the last writer.
func StopBatchUpdater() {
	batchUpdaterLock.Lock()
	cancel, done := batchUpdaterCancel, batchUpdaterDone
	batchUpdaterCancel, batchUpdaterDone = nil, nil
	batchUpdaterLock.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

func addNewRecord(type_ int, id int, value int) {
	batchUpdateLocks[type_].Lock()
	defer batchUpdateLocks[type_].Unlock()
	old, ok := batchUpdateStores[type_][id]
	if !ok {
		batchUpdateStores[type_][id] = value
		return
	}

	sum := old + value
	if (value > 0 && sum < old) || (value < 0 && sum > old) {
		common.SysError(common.LogText("batch update overflow: type=%d id=%d old=%d value=%d", type_, id, old, value))
		if value > 0 {
			sum = math.MaxInt
		} else {
			sum = math.MinInt
		}
	}
	batchUpdateStores[type_][id] = sum
}

// BatchFlushResult counts the user, token and channel updates one
// FlushBatchUpdates call wrote, and the writes that failed (a failed delta is
// logged and dropped, as before).
type BatchFlushResult struct {
	Users    int
	Tokens   int
	Channels int
	Failed   int
}

// FlushBatchUpdates writes every delta pending in the batch stores to the
// database once. Deltas added while it runs stay in the stores for the next
// flush. Safe to call concurrently with the periodic updater.
func FlushBatchUpdates() BatchFlushResult {
	batchFlushLock.Lock()
	defer batchFlushLock.Unlock()

	var result BatchFlushResult
	// check if there's any data to update
	hasData := false
	for i := range BatchUpdateTypeCount {
		batchUpdateLocks[i].Lock()
		if len(batchUpdateStores[i]) > 0 {
			hasData = true
			batchUpdateLocks[i].Unlock()
			break
		}
		batchUpdateLocks[i].Unlock()
	}

	if !hasData {
		return result
	}

	common.SysLog(common.LogText("batch update started"))
	stores := make([]map[int]int, BatchUpdateTypeCount)
	for i := range BatchUpdateTypeCount {
		batchUpdateLocks[i].Lock()
		stores[i] = batchUpdateStores[i]
		batchUpdateStores[i] = make(map[int]int)
		batchUpdateLocks[i].Unlock()
	}

	for i, store := range stores {
		if i == BatchUpdateTypeUserQuota || i == BatchUpdateTypeUsedQuota || i == BatchUpdateTypeRequestCount {
			continue
		}
		for key, value := range store {
			switch i {
			case BatchUpdateTypeTokenQuota:
				err := increaseTokenQuota(key, value)
				if err != nil {
					common.SysLog(common.LogText("failed to batch update token quota: %s", err.Error()))
					result.Failed++
				} else {
					result.Tokens++
				}
			case BatchUpdateTypeChannelUsedQuota:
				if updateChannelUsedQuota(key, value) != nil {
					result.Failed++
				} else {
					result.Channels++
				}
			}
		}
	}

	userQuotaStore := stores[BatchUpdateTypeUserQuota]
	usedQuotaStore := stores[BatchUpdateTypeUsedQuota]
	requestCountStore := stores[BatchUpdateTypeRequestCount]

	userIDs := make(map[int]struct{}, len(userQuotaStore)+len(usedQuotaStore)+len(requestCountStore))
	for key := range userQuotaStore {
		userIDs[key] = struct{}{}
	}
	for key := range usedQuotaStore {
		userIDs[key] = struct{}{}
	}
	for key := range requestCountStore {
		userIDs[key] = struct{}{}
	}
	for key := range userIDs {
		if userQuotaStore[key] == 0 && usedQuotaStore[key] == 0 && requestCountStore[key] == 0 {
			continue
		}
		if updateUserQuotaUsedQuotaAndRequestCount(key, userQuotaStore[key], usedQuotaStore[key], requestCountStore[key]) != nil {
			result.Failed++
		} else {
			result.Users++
		}
	}
	common.SysLog(common.LogText("batch update finished"))
	return result
}

func RecordExist(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func shouldUpdateRedis(fromDB bool, err error) bool {
	return common.RedisEnabled && fromDB && err == nil
}
