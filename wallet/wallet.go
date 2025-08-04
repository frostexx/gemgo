package wallet

import (
	"fmt"
	"os"
	"pi/util"
	"strconv"

	"github.com/stellar/go/clients/horizonclient"
	hClient "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/txnbuild"
)

type Wallet struct {
	networkPassphrase string
	serverURL         string
	client            *hClient.Client
	baseReserve       float64
}

func New() *Wallet {
	client := hClient.DefaultPublicNetClient
	client.HorizonURL = os.Getenv("NET_URL")

	w := &Wallet{
		networkPassphrase: os.Getenv("NET_PASSPHRASE"),
		serverURL:         os.Getenv("NET_URL"),
		client:            client,
		baseReserve:       0.49,
	}
	w.GetBaseReserve()

	return w
}

func (w *Wallet) GetBaseReserve() {
	ledger, err := w.client.Ledgers(horizonclient.LedgerRequest{Order: horizonclient.OrderDesc, Limit: 1})
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(ledger.Embedded.Records) == 0 {
		fmt.Println(err)
		return
	}

	baseReserveStr := ledger.Embedded.Records[0].BaseReserve
	w.baseReserve = float64(baseReserveStr) / 1e7
	fmt.Println(w.baseReserve)
}

func (w *Wallet) GetAddress(kp *keypair.Full) string {
	return kp.Address()
}

func (w *Wallet) Login(seedPhrase string) (*keypair.Full, error) {
	kp, err := util.GetKeyFromSeed(seedPhrase)
	if err != nil {
		return nil, err
	}

	return kp, nil
}

func (w *Wallet) GetAccount(kp *keypair.Full) (horizon.Account, error) {
	accReq := hClient.AccountRequest{AccountID: kp.Address()}
	account, err := w.client.AccountDetail(accReq)
	if err != nil {
		return horizon.Account{}, fmt.Errorf("error fetching account details: %v", err)
	}

	return account, nil
}

func (w *Wallet) GetAvailableBalance(kp *keypair.Full) (string, error) {
	account, err := w.GetAccount(kp)
	if err != nil {
		return "", err
	}

	var totalBalance float64
	for _, b := range account.Balances {
		if b.Type == "native" {
			totalBalance, err = strconv.ParseFloat(b.Balance, 64)
			if err != nil {
				return "", err
			}
			break
		}
	}

	reserve := w.baseReserve * float64(2+account.SubentryCount)
	available := totalBalance - reserve
	if available < 0 {
		available = 0
	}

	availableStr := fmt.Sprintf("%.2f", available)

	return availableStr, nil
}

func (w *Wallet) GetTransactions(kp *keypair.Full, limit uint) ([]operations.Operation, error) {
	opReq := hClient.OperationRequest{
		ForAccount: kp.Address(),
		Limit:      limit,
		Order:      hClient.OrderDesc,
	}
	ops, err := w.client.Operations(opReq)
	if err != nil {
		return nil, fmt.Errorf("error fetching account operations: %v", err)
	}

	return ops.Embedded.Records, nil
}

func (w *Wallet) GetLockedBalances(kp *keypair.Full) ([]horizon.ClaimableBalance, error) {
	cbReq := hClient.ClaimableBalanceRequest{
		Claimant: kp.Address(),
	}
	cbs, err := w.client.ClaimableBalances(cbReq)
	if err != nil {
		return nil, fmt.Errorf("error fetching claimable balances: %v", err)
	}

	return cbs.Embedded.Records, nil
}

func (w *Wallet) GetClaimableBalance(balanceID string) (horizon.ClaimableBalance, error) {
	cb, err := w.client.ClaimableBalance(balanceID)
	if err != nil {
		return horizon.ClaimableBalance{}, fmt.Errorf("error fetching claimable balance: %v", err)
	}

	return cb, nil
}

// Enhanced transfer method with custom fee
func (w *Wallet) TransferWithFee(kp *keypair.Full, amountStr string, address string, customFee int64) error {
	w.GetBaseReserve()
	baseReserve := w.baseReserve

	// Parse requested amount
	requestedAmount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	// Get account details
	account, err := w.GetAccount(kp)
	if err != nil {
		return fmt.Errorf("error getting account: %w", err)
	}

	// Get actual native (PI) balance
	var nativeBalance float64
	for _, bal := range account.Balances {
		if bal.Asset.Type == "native" {
			nativeBalance, err = strconv.ParseFloat(bal.Balance, 64)
			if err != nil {
				return fmt.Errorf("invalid balance format: %w", err)
			}
			break
		}
	}

	// Calculate minimum required balance
	minBalance := baseReserve * float64(2+account.SubentryCount)

	// Available balance = total - reserve - custom fee
	feeInPI := float64(customFee) / 1e7
	available := nativeBalance - minBalance - feeInPI

	if available <= 0 {
		return fmt.Errorf("insufficient available balance")
	}

	requestedAmount = available - 0.01

	// Ensure requested amount is transferable
	if requestedAmount > available {
		return fmt.Errorf("requested amount %.7f exceeds available balance %.7f", requestedAmount, available)
	}

	// Build payment operation
	paymentOp := &txnbuild.Payment{
		Destination:   address,
		Amount:        fmt.Sprintf("%.7f", requestedAmount),
		Asset:         txnbuild.NativeAsset{},
		SourceAccount: kp.Address(),
	}

	// Build transaction with custom fee
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &account,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{paymentOp},
			BaseFee:              customFee,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewInfiniteTimeout(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error building transaction: %w", err)
	}

	// Sign transaction
	tx, err = tx.Sign(w.networkPassphrase, kp)
	if err != nil {
		return fmt.Errorf("error signing transaction: %w", err)
	}

	// Submit transaction
	resp, err := w.client.SubmitTransaction(tx)
	if err != nil {
		if resp != nil && resp.Extras != nil {
			return getTxErrorFromResultXdr(resp.Extras.ResultXdr)
		}
		return fmt.Errorf("error submitting transaction: %w", err)
	}

	return nil
}

// Enhanced claim method with custom fee
func (w *Wallet) ClaimBalance(kp *keypair.Full, balanceID string, customFee int64) error {
	account, err := w.GetAccount(kp)
	if err != nil {
		return fmt.Errorf("error getting account: %w", err)
	}

	claimOp := &txnbuild.ClaimClaimableBalance{
		BalanceID: balanceID,
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &account,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{claimOp},
			BaseFee:              customFee,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewInfiniteTimeout(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error building transaction: %w", err)
	}

	tx, err = tx.Sign(w.networkPassphrase, kp)
	if err != nil {
		return fmt.Errorf("error signing transaction: %w", err)
	}

	resp, err := w.client.SubmitTransaction(tx)
	if err != nil {
		if resp != nil && resp.Extras != nil {
			return getTxErrorFromResultXdr(resp.Extras.ResultXdr)
		}
		return fmt.Errorf("error submitting transaction: %w", err)
	}

	return nil
}