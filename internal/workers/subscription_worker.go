package workers

import (
	"time"
	"log"
)

func StartSubscriptionWorker() {

	go func() {
		for {

			ProcessRenewals()

			time.Sleep(1 * time.Hour)
		}
	}()
}
