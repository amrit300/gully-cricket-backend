package workers

import (
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"gully-cricket/internal/queue"
)

//////////////////////////////////////////////////////////////
// CONFIG (AUTO SCALE BASED ON CPU)
//////////////////////////////////////////////////////////////

var WorkerCount = runtime.NumCPU() * 2

//////////////////////////////////////////////////////////////
// METRICS (OBSERVABILITY)
//////////////////////////////////////////////////////////////

var (
	ActiveWorkers uint64
	ProcessedJobs uint64
	FailedJobs    uint64
)

//////////////////////////////////////////////////////////////
// START WORKER POOL
//////////////////////////////////////////////////////////////

func StartWorkerPool() {

	log.Printf("🚀 Starting worker pool with %d workers\n", WorkerCount)

	for i := 0; i < WorkerCount; i++ {
		go worker(i)
	}
}

//////////////////////////////////////////////////////////////
// WORKER
//////////////////////////////////////////////////////////////

func worker(id int) {

	log.Printf("Worker %d started\n", id)

	for job := range queue.JobQueue {

		atomic.AddUint64(&ActiveWorkers, 1)

		start := time.Now()

		err := processJob(job)

		if err != nil {
			log.Printf("❌ Worker %d failed job=%s err=%v\n", id, job.Type, err)

			atomic.AddUint64(&FailedJobs, 1)

			// retry logic
			queue.Retry(job)

		} else {
			atomic.AddUint64(&ProcessedJobs, 1)
		}

		duration := time.Since(start)

		if duration > 2*time.Second {
			log.Printf("⚠️ Slow job: %s took %v\n", job.Type, duration)
		}

		atomic.AddUint64(&ActiveWorkers, ^uint64(0)) // decrement
	}
}

//////////////////////////////////////////////////////////////
// JOB ROUTER (CORE LOGIC)
//////////////////////////////////////////////////////////////

func processJob(job queue.Job) error {

	switch job.Type {

	case "leaderboard_update":
		return handleLeaderboard(job.Data)

	case "match_complete":
		return handleMatchComplete(job.Data)

	default:
		log.Println("⚠️ Unknown job type:", job.Type)
		return nil
	}
}

//////////////////////////////////////////////////////////////
// STATS (FOR MONITORING)
//////////////////////////////////////////////////////////////

func Stats() map[string]uint64 {

	return map[string]uint64{
		"active_workers": atomic.LoadUint64(&ActiveWorkers),
		"processed_jobs": atomic.LoadUint64(&ProcessedJobs),
		"failed_jobs":    atomic.LoadUint64(&FailedJobs),
		"queue_length":   uint64(len(queue.JobQueue)),
	}
}
