package workers

import (
	"log"

	"gully-cricket/internal/queue"
)

//////////////////////////////////////////////////////////////
// START WORKERS (SHARDED QUEUES)
//////////////////////////////////////////////////////////////

func StartWorkerPool(workerCount int) {

	for i := 0; i < queue.NumQueues; i++ {

		go func(q chan queue.Job) {

			for job := range q {
				processJob(job)
			}

		}(queue.Queues[i])
	}

	log.Println("✅ Worker pool started")
}

//////////////////////////////////////////////////////////////
// JOB ROUTER
//////////////////////////////////////////////////////////////

func processJob(job queue.Job) {

	switch job.Type {

	case "leaderboard_update":
		err := handleLeaderboard(job.Data)
		if err != nil {
			queue.Retry(job)
		}

	case "fraud_check":
		handleFraudCheck(job.Data)

	default:
		log.Println("Unknown job:", job.Type)
	}
}
