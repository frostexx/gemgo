package core

import (
	"log"
	"pi/server/network"
	"sync"
	"time"
)

type TransferWorker struct {
	config       BotConfig
	client       *network.PiClient
	wg           *sync.WaitGroup
	updateStatus func(string, string)
}

func NewTransferWorker(config BotConfig, client *network.PiClient, wg *sync.WaitGroup, updateStatus func(string, string)) *TransferWorker {
	return &TransferWorker{
		config:       config,
		client:       client,
		wg:           wg,
		updateStatus: updateStatus,
	}
}

// Run starts the transfer process. It does not depend on the claim status.
func (w *TransferWorker) Run(stopChan <-chan struct{}) {
	defer w.wg.Done()
	w.updateStatus("TransferWorker", "Starting transfer attempts...")
	log.Println("TransferWorker: Initiating transfer sequence.")

	// This worker can try to transfer unlocked funds.
	// We assume it knows the destination address from config.
	destinationAddress := "DESTINATION_WALLET_ADDRESS" // This should be in config
	fee := int64(3200000) // Use a standard fee for transfer, or make it adaptive too.

	maxRetries := 10
	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-stopChan:
			log.Println("TransferWorker: Stop signal received. Aborting.")
			return
		default:
			w.updateStatus("TransferWorker", "Submitting transfer transaction...")
			// This would build and submit a transfer transaction.
			err := w.client.SubmitTransaction("transfer", w.config.MainWalletSeed, w.config.SponsorSeed, fee)

			if err == nil {
				log.Printf("SUCCESS! TransferWorker succeeded on attempt %d.", attempt+1)
				w.updateStatus("TransferWorker", "Transfer Succeeded!")
				return // Success, exit
			}

			log.Printf("TransferWorker [Attempt %d/%d]: Failed. Error: %v", attempt+1, maxRetries, err)
			time.Sleep(200 * time.Millisecond) // Simple retry delay for transfer
		}
	}

	log.Println("TransferWorker: Failed to transfer after all attempts.")
	w.updateStatus("TransferWorker", "Finished all transfer attempts (failed).")
}