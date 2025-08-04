package wallet

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stellar/go/keypair"
)

type ConcurrentProcessor struct {
	wallet *Wallet
	sponsor *SponsorWallet
	flooder *NetworkFlooder
	config *Config
}

func NewConcurrentProcessor(wallet *Wallet, sponsor *SponsorWallet, config *Config) *ConcurrentProcessor {
	return &ConcurrentProcessor{
		wallet: wallet,
		sponsor: sponsor,
		flooder: NewNetworkFlooder(wallet, config),
		config: config,
	}
}

func (cp *ConcurrentProcessor) ExecuteConcurrentOperations(
	ctx context.Context,
	mainKp *keypair.Full,
	claimableBalanceID string,
	withdrawalAddress string,
	unlockTime time.Time,
) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// 1. Start network flooding
	wg.Add(1)
	go func() {
		defer wg.Done()
		cp.flooder.FloodNetwork(ctx, mainKp, unlockTime)
	}()

	// 2. Execute claiming at unlock time
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cp.executeClaiming(ctx, mainKp, claimableBalanceID, unlockTime); err != nil {
			errChan <- fmt.Errorf("claiming failed: %w", err)
		}
	}()

	// 3. Execute transfer independently at unlock time
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cp.executeTransfer(ctx, mainKp, withdrawalAddress, unlockTime); err != nil {
			errChan <- fmt.Errorf("transfer failed: %w", err)
		}
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("concurrent operations had errors: %v", errors)
	}

	return nil
}

func (cp *ConcurrentProcessor) executeClaiming(ctx context.Context, kp *keypair.Full, balanceID string, unlockTime time.Time) error {
	timer := time.NewTimer(time.Until(unlockTime))
	defer timer.Stop()

	select {
	case <-timer.C:
		return cp.executeMultipleClaimAttempts(ctx, kp, balanceID)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (cp *ConcurrentProcessor) executeMultipleClaimAttempts(ctx context.Context, kp *keypair.Full, balanceID string) error {
	semaphore := make(chan struct{}, cp.config.MaxConcurrentClaims)
	var wg sync.WaitGroup
	var successOnce sync.Once
	var success bool

	for i := 0; i < cp.config.MaxRetries; i++ {
		wg.Add(1)
		go func(attempt int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			competitiveFee := GetCompetitiveFee(cp.config.ClaimingFee, true)
			
			if cp.sponsor != nil {
				err := cp.sponsor.SponsorClaim(kp, balanceID, competitiveFee)
				if err == nil {
					successOnce.Do(func() { success = true })
				}
			} else {
				err := cp.wallet.ClaimBalance(kp, balanceID, competitiveFee)
				if err == nil {
					successOnce.Do(func() { success = true })
				}
			}

			time.Sleep(time.Duration(cp.config.RetryDelay) * time.Millisecond)
		}(i)
	}

	wg.Wait()

	if !success {
		return fmt.Errorf("all claiming attempts failed")
	}
	return nil
}

func (cp *ConcurrentProcessor) executeTransfer(ctx context.Context, kp *keypair.Full, address string, unlockTime time.Time) error {
	timer := time.NewTimer(time.Until(unlockTime))
	defer timer.Stop()

	select {
	case <-timer.C:
		return cp.executeMultipleTransferAttempts(ctx, kp, address)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (cp *ConcurrentProcessor) executeMultipleTransferAttempts(ctx context.Context, kp *keypair.Full, address string) error {
	semaphore := make(chan struct{}, cp.config.MaxConcurrentTransfers)
	var wg sync.WaitGroup

	for i := 0; i < cp.config.MaxRetries; i++ {
		wg.Add(1)
		go func(attempt int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			balance, _ := cp.wallet.GetAvailableBalance(kp)
			competitiveFee := GetCompetitiveFee(cp.config.TransferFee, false)
			
			cp.wallet.TransferWithFee(kp, balance, address, competitiveFee)
			
			time.Sleep(time.Duration(cp.config.RetryDelay) * time.Millisecond)
		}(i)
	}

	wg.Wait()
	return nil
}