package core

import (
	"errors"
	"fmt"
	"log"
	"pi/server/network"
	"sync"
	"time"
)

type BotConfig struct {
	NetworkURL      string
	NetworkPass     string
	MainWalletSeed  string
	SponsorSeed     string
	UnlockTimestamp string
}

type Bot struct {
	config      BotConfig
	client      *network.PiClient
	taskManager *TaskManager
	status      string
	statusMux   sync.RWMutex
	stopChan    chan struct{}
	running     bool
	runningMux  sync.Mutex
}

func NewBot(config BotConfig) (*Bot, error) {
	if config.MainWalletSeed == "" || config.SponsorSeed == "" {
		return nil, errors.New("MAIN_WALLET_SEED and SPONSOR_WALLET_SEED must be set")
	}

	client, err := network.NewPiClient(config.NetworkURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pi client: %w", err)
	}

	return &Bot{
		config:   config,
		client:   client,
		status:   "Idle",
		stopChan: make(chan struct{}),
	}, nil
}

func (b *Bot) Run() {
	log.Println("Bot is running in the background, waiting for start command.")
	// In a real scenario, this could run background health checks, etc.
	// For now, it just keeps the bot alive.
	<-b.stopChan
	log.Println("Bot has been shut down.")
}

func (b *Bot) Stop() {
	b.StopOperations()
	close(b.stopChan)
}

func (b *Bot) StartOperations() error {
	b.runningMux.Lock()
	defer b.runningMux.Unlock()

	if b.running {
		return errors.New("bot is already running")
	}

	unlockTime, err := time.Parse(time.RFC3339, b.config.UnlockTimestamp)
	if err != nil {
		return fmt.Errorf("invalid UNLOCK_TIMESTAMP format: %w. Use RFC3339 format (e.g., 2023-10-27T10:00:00Z)", err)
	}

	b.taskManager = NewTaskManager(b.config, b.client)
	
	log.Printf("Operations starting. Targeting unlock time: %s", unlockTime.String())
	b.setStatus("Running: Waiting for unlock time.")
	
	go b.taskManager.Start(unlockTime)
	b.running = true
	
	return nil
}

func (b *Bot) StopOperations() {
	b.runningMux.Lock()
	defer b.runningMux.Unlock()

	if !b.running {
		return
	}

	if b.taskManager != nil {
		b.taskManager.Stop()
	}
	log.Println("Operations stopped by user.")
	b.setStatus("Idle: Stopped by user.")
	b.running = false
}

func (b *Bot) setStatus(s string) {
	b.statusMux.Lock()
	defer b.statusMux.Unlock()
	b.status = s
	log.Println("Bot Status:", s)
}

func (b *Bot) GetStatus() map[string]interface{} {
	b.statusMux.RLock()
	defer b.statusMux.RUnlock()
	
	status := map[string]interface{}{
		"bot_status": b.status,
	}

	if b.taskManager != nil {
		for k, v := range b.taskManager.GetStatus() {
			status[k] = v
		}
	}
	return status
}