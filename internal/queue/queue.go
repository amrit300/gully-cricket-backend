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
	NumQueues       = 16           // 🔥 sharded queues
	QueueSize       = 20000       // per queue → total 320K buffer
	MaxRetry        = 3
	EnqueueTimeout  = 20 * time.Millisecond
	RetryDelay      = 100 * time.Millisecond
)

//////////////////////////////////////////////////////////////
// JOB STRUCT (ADD PRIORITY + KEY)
//////////////////////////////////////////////////////////////

type Job struct {
	Type      string
	Data      interface{}
	Retry     int
	Priority  int    // 0 = high, 1 = normal
	Key       string // for sharding (user_id / contest_id)
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
	q := Queues[idx]

  if len(q) > QueueSize*90/100 {
	log.Printf("⚠️ Queue shard=%d 80%% full (%d/%d)\n", idx, len(q), QueueSize)
}

	select {

	// FAST PATH
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
