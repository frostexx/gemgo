package core

import (
	"log"
	"pi/server/network"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskManager struct {
	config      BotConfig
	client      *network.PiClient
	claimWorker *ClaimWorker
	transWorker *TransferWorker
	stopChan    chan struct{}
	wg          sync.WaitGroup
	status      map[string]string
	statusMux   sync.RWMutex
}

func NewTaskManager(config BotConfig, client *network.PiClient) *TaskManager {
	return &TaskManager{
		config:   config,
		client:   client,
		stopChan: make(chan struct{}),
		status:   make(map[string]string),
	}
}

func (tm *TaskManager) Start(unlockTime time.Time) {
	log.Printf("Task Manager started. Waiting until unlock time: %v", unlockTime)

	// High-precision timer
	timer := time.NewTimer(time.Until(unlockTime))
	defer timer.Stop()

	select {
	case <-timer.C:
		log.Println("UNLOCK TIME REACHED! Unleashing workers.")
		tm.runWorkers()
	case <-tm.stopChan:
		log.Println("Task Manager stopped before unlock time.")
		return
	}
}

func (tm *TaskManager) runWorkers() {
	tm.claimWorker = NewClaimWorker(tm.config, tm.client, &tm.wg, tm.updateStatus)
	tm.transWorker = NewTransferWorker(tm.config, tm.client, &tm.wg, tm.updateStatus)

	tm.wg.Add(2)
	go tm.claimWorker.Run(tm.stopChan)
	go tm.transWorker.Run(tm.stopChan)

	// Wait for workers to finish or be stopped
	tm.wg.Wait()
	log.Println("All workers have completed their tasks.")
}

func (tm *TaskManager) Stop() {
	close(tm.stopChan)
}

func (tm *TaskManager) updateStatus(workerName, status string) {
	tm.statusMux.Lock()
	defer tm.statusMux.Unlock()
	tm.status[workerName] = status
}

func (tm *TaskManager) GetStatus() gin.H {
	tm.statusMux.RLock()
	defer tm.statusMux.RUnlock()
	
	s := gin.H{}
	for k, v := range tm.status {
		s[k] = v
	}
	return s
}