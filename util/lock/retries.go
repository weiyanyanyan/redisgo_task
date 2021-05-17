package lock

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	//默认为4次，重试间隔数值最大为15
	MAX_COUNT  = 4
	//默认从0开始
	INIT_COUNT = 0
)

type Retry struct {
	Count           float64
	MaxRetriedCount float64
	MonitorRetryAll bool
	L               sync.Mutex
}

func NewRetry(count float64, maxRetriedCount float64, monitorRetryAll bool) *Retry {
	if count <= 0 {
		count = INIT_COUNT
	}
	if maxRetriedCount <= 0 {
		maxRetriedCount = MAX_COUNT
	}
	return &Retry{
		Count:           count,
		MaxRetriedCount: maxRetriedCount,
		MonitorRetryAll: monitorRetryAll,
	}
}

func (r *Retry) RetriesTime() int {
	r.L.Lock()
	defer r.L.Unlock()
	if r.Count < 0 {
		return -1
	}
	if r.Count > r.MaxRetriedCount {
		if r.MonitorRetryAll {
			r.Count = 0
		} else {
			return -1
		}
	}
	retriesTime := RandNumber(r.Count)
	r.Count++
	return retriesTime
}
/*
重试机制遵循二进制退避原则，保证重试命中率及服务稳定性
 */
func RandNumber(n float64) int {
	randNum := 0
	index := math.Pow(2, n)
	MaxNum := int(index) - 1
	if MaxNum <= 0 {
		return 0
	}
	rand.Seed(time.Now().UnixNano())
	randNum = rand.Intn(MaxNum)
	return randNum
}
