package wallet

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

var ErrUnAuthorized = errors.New("unauthorized")

func getTxErrorFromResultXdr(resultXdr string) error {
	var txResult xdr.TransactionResult
	if err := xdr.SafeUnmarshalBase64(resultXdr, &txResult); err != nil {
		return fmt.Errorf("failed to decode result XDR: %w", err)
	}

	// Transaction-level error
	if txResult.Result.Code != xdr.TransactionResultCodeTxSuccess {
		return fmt.Errorf("transaction failed with code: %s", txResult.Result.Code.String())
	}

	if txResult.Result.Results == nil {
		return fmt.Errorf("transaction succeeded but no operation results returned")
	}

	for i, opResult := range *txResult.Result.Results {
		switch opResult.Tr.Type {
		case xdr.OperationTypePayment:
			if opResult.Tr.PaymentResult == nil {
				return fmt.Errorf("operation %d: missing payment result", i)
			}
			code := opResult.Tr.PaymentResult.Code
			if code != xdr.PaymentResultCodePaymentSuccess {
				return fmt.Errorf("operation %d failed: %s", i, code.String())
			}

		case xdr.OperationTypeClaimClaimableBalance:
			if opResult.Tr.ClaimClaimableBalanceResult == nil {
				return fmt.Errorf("operation %d: missing claim claimable balance result", i)
			}
			code := opResult.Tr.ClaimClaimableBalanceResult.Code
			if code != xdr.ClaimClaimableBalanceResultCodeClaimClaimableBalanceSuccess {
				return fmt.Errorf("operation %d failed: %s", i, code.String())
			}

		default:
			return fmt.Errorf("operation %d has unsupported type: %s", i, opResult.Tr.Type.String())
		}
	}

	return nil
}

func (w *Wallet) Transfer(kp *keypair.Full, amountStr string, address string) error {
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

	// Available balance = total - reserve - 1 base fee
	available := nativeBalance - minBalance - 0.00001
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

	// Build transaction
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &account,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{paymentOp},
			BaseFee:              100,
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