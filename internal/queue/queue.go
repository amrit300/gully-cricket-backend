package queue

import (
	"log"
	"sync/atomic"
	"time"
)

//////////////////////////////////////////////////////////////
// CONFIG (TUNED FOR BURST TRAFFIC)
//////////////////////////////////////////////////////////////

const (
	NumQueues      = 16
	QueueSize      = 20000
	MaxRetry       = 3
	EnqueueTimeout = 20 * time.Millisecond
	RetryDelay     = 100 * time.Millisecond
)

//////////////////////////////////////////////////////////////
// JOB STRUCT (ADD PRIORITY + KEY)
//////////////////////////////////////////////////////////////

type Job struct {
	Type      string
	Data      interface{}
	Retry     int
	Priority  int
	Key       string
	CreatedAt time.Time
}

//////////////////////////////////////////////////////////////
// SHARDED QUEUES
//////////////////////////////////////////////////////////////

var Queues [NumQueues]chan Job

func Init() {
	for i := 0; i < NumQueues; i++ {
		Queues[i] = make(chan Job, QueueSize)
	}
}

//////////////////////////////////////////////////////////////
// METRICS
//////////////////////////////////////////////////////////////

var (
	TotalEnqueued uint64
	TotalDropped  uint64
	TotalRetried  uint64
)

//////////////////////////////////////////////////////////////
// SHARD PICKER (CONSISTENT HASH)
//////////////////////////////////////////////////////////////

func getQueueIndex(key string) int {
	if key == "" {
		return 0
	}

	hash := 0
	for i := 0; i < len(key); i++ {
		hash += int(key[i])
	}
	return hash % NumQueues
}

//////////////////////////////////////////////////////////////
// ENQUEUE (BURST SAFE)
//////////////////////////////////////////////////////////////

func Enqueue(job Job) {

	job.CreatedAt = time.Now()

	idx := getQueueIndex(job.Key)

	// 🔥 PRIORITY OVERRIDE (HIGH PRIORITY → shard 0)
	if job.Priority == 0 {
		idx = 0
	}

	q := Queues[idx]

	// 🔥 SAFETY CHECK (INIT NOT CALLED)
	if q == nil {
		log.Println("QUEUE NOT INITIALIZED")
		return
	}

	// 🔥 80% ALERT
	if len(q) > QueueSize*90/100 {
		log.Printf("⚠️ Queue shard=%d 90%% full (%d/%d)\n", idx, len(q), QueueSize)
	}

	select {

	case q <- job:
		atomic.AddUint64(&TotalEnqueued, 1)
		return

	default:
		timer := time.NewTimer(EnqueueTimeout)
		defer timer.Stop()

		select {
		case q <- job:
			atomic.AddUint64(&TotalEnqueued, 1)

		case <-timer.C:
			atomic.AddUint64(&TotalDropped, 1)
			log.Printf("QUEUE FULL shard=%d job=%s\n", idx, job.Type)
		}
	}
}

//////////////////////////////////////////////////////////////
// RETRY
//////////////////////////////////////////////////////////////

func Retry(job Job) {

	if job.Retry >= MaxRetry {
		log.Printf("JOB FAILED permanently: %s\n", job.Type)
		return
	}

	job.Retry++
	atomic.AddUint64(&TotalRetried, 1)

	time.AfterFunc(RetryDelay, func() {
		Enqueue(job)
	})
}

//////////////////////////////////////////////////////////////
// STATS
//////////////////////////////////////////////////////////////

func Stats() map[string]uint64 {
	return map[string]uint64{
		"enqueued": atomic.LoadUint64(&TotalEnqueued),
		"dropped":  atomic.LoadUint64(&TotalDropped),
		"retried":  atomic.LoadUint64(&TotalRetried),
	}
}
