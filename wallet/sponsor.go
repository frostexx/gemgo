package wallet

import (
	"fmt"
	"pi/util"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
)

type SponsorWallet struct {
	keyPair *keypair.Full
	wallet  *Wallet
}

func NewSponsorWallet(seedPhrase string, wallet *Wallet) (*SponsorWallet, error) {
	kp, err := util.GetKeyFromSeed(seedPhrase)
	if err != nil {
		return nil, fmt.Errorf("invalid sponsor seed phrase: %w", err)
	}

	return &SponsorWallet{
		keyPair: kp,
		wallet:  wallet,
	}, nil
}

func (sw *SponsorWallet) GetAddress() string {
	return sw.keyPair.Address()
}

func (sw *SponsorWallet) SponsorClaim(mainWallet *keypair.Full, claimableBalanceID string, competitiveFee int64) error {
	// Get accounts
	sponsorAccount, err := sw.wallet.GetAccount(sw.keyPair)
	if err != nil {
		return fmt.Errorf("error getting sponsor account: %w", err)
	}

	// Build sponsored transaction
	claimOp := &txnbuild.ClaimClaimableBalance{
		BalanceID: claimableBalanceID,
	}

	// Create sponsored transaction
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &sponsorAccount,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{claimOp},
			BaseFee:              competitiveFee,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewInfiniteTimeout(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error building sponsored transaction: %w", err)
	}

	// Sign with sponsor first, then main wallet
	tx, err = tx.Sign(sw.wallet.networkPassphrase, sw.keyPair, mainWallet)
	if err != nil {
		return fmt.Errorf("error signing sponsored transaction: %w", err)
	}

	// Submit transaction
	_, err = sw.wallet.client.SubmitTransaction(tx)
	if err != nil {
		return fmt.Errorf("error submitting sponsored claim: %w", err)
	}

	return nil
}