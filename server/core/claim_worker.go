package core

import (
	"log"
	"pi/server/network"
	"sync"
	"time"
)

type ClaimWorker struct {
	config       BotConfig
	client       *network.PiClient
	wg           *sync.WaitGroup
	updateStatus func(string, string)
}

func NewClaimWorker(config BotConfig, client *network.PiClient, wg *sync.WaitGroup, updateStatus func(string, string)) *ClaimWorker {
	return &ClaimWorker{
		config:       config,
		client:       client,
		wg:           wg,
		updateStatus: updateStatus,
	}
}

func (w *ClaimWorker) Run(stopChan <-chan struct{}) {
	defer w.wg.Done()
	w.updateStatus("ClaimWorker", "Starting claim attempts...")
	log.Println("ClaimWorker: Initiating claim sequence.")

	// Example of multi-vector attack: try with 3 different fee strategies concurrently
	feeStrategies := []int64{3200000, 9400000, 15000000} // Low, Medium, High fees
	var successOnce sync.Once
	var claimWg sync.WaitGroup

	for i, fee := range feeStrategies {
		claimWg.Add(1)
		go func(strategyIndex int, transactionFee int64) {
			defer claimWg.Done()
			
			// Each strategy gets its own retry logic
			maxRetries := 5
			for attempt := 0; attempt < maxRetries; attempt++ {
				select {
				case <-stopChan:
					log.Printf("ClaimWorker [Strategy %d]: Stop signal received. Aborting.", strategyIndex)
					return
				default:
					w.updateStatus("ClaimWorker", "Submitting claim transaction...")
					// In a real implementation, this would build and submit a claim transaction
					// For now, we simulate the logic.
					err := w.client.SubmitTransaction("claim", w.config.MainWalletSeed, w.config.SponsorSeed, transactionFee)

					if err == nil {
						successOnce.Do(func() {
							log.Printf("SUCCESS! ClaimWorker [Strategy %d] succeeded with fee %d", strategyIndex, transactionFee)
							w.updateStatus("ClaimWorker", "Claim Succeeded!")
							// In a real scenario, signal other goroutines to stop.
						})
						return // This goroutine's job is done
					}

					log.Printf("ClaimWorker [Strategy %d, Attempt %d/%d]: Failed. Error: %v", strategyIndex, attempt+1, maxRetries, err)
					
					// Exponential backoff with jitter
					backoff := time.Duration(100+50*attempt) * time.Millisecond
					time.Sleep(backoff)
				}
			}
		}(i, fee)
	}

	claimWg.Wait()
	log.Println("ClaimWorker: All claim strategies have completed.")
	w.updateStatus("ClaimWorker", "Finished all claim attempts.")
}