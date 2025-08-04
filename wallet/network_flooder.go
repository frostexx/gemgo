package wallet

import (
	"context"
	"pi/config"
	"sync"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
)

type NetworkFlooder struct {
	wallet *Wallet
	config *config.Config
}

func NewNetworkFlooder(wallet *Wallet, cfg *config.Config) *NetworkFlooder {
	return &NetworkFlooder{
		wallet: wallet,
		config: cfg,
	}
}

func (nf *NetworkFlooder) FloodNetwork(ctx context.Context, kp *keypair.Full, unlockTime time.Time) {
	// Start flooding 200ms before unlock time
	floodStart := unlockTime.Add(-200 * time.Millisecond)
	
	timer := time.NewTimer(time.Until(floodStart))
	defer timer.Stop()

	select {
	case <-timer.C:
		nf.executeFlood(ctx, kp)
	case <-ctx.Done():
		return
	}
}

func (nf *NetworkFlooder) executeFlood(ctx context.Context, kp *keypair.Full) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, nf.config.FloodingGoroutines)

	for i := 0; i < nf.config.FloodingGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			nf.sendFloodTransaction(ctx, kp)
		}()
	}

	wg.Wait()
}

func (nf *NetworkFlooder) sendFloodTransaction(ctx context.Context, kp *keypair.Full) {
	account, err := nf.wallet.GetAccount(kp)
	if err != nil {
		return
	}

	// Create a minimal operation to flood the network
	bumpOp := &txnbuild.BumpSequence{
		BumpTo: int64(account.Sequence) + 1,
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &account,
			IncrementSequenceNum: false, // Don't increment for flooding
			Operations:           []txnbuild.Operation{bumpOp},
			BaseFee:              1000000, // 0.1 PI
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewTimeout(300),
			},
		},
	)
	if err != nil {
		return
	}

	tx, err = tx.Sign(nf.wallet.networkPassphrase, kp)
	if err != nil {
		return
	}

	// Submit and ignore errors (flooding purpose)
	nf.wallet.client.SubmitTransaction(tx)
}