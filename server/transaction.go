package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"pi/config"
	"pi/util"
	"pi/wallet"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
)

type WithdrawRequest struct {
	SeedPhrase        string `json:"seed_phrase"`
	SponsorSeedPhrase string `json:"sponsor_seed_phrase,omitempty"`
	LockedBalanceID   string `json:"locked_balance_id"`
	WithdrawalAddress string `json:"withdrawal_address"`
	Amount            string `json:"amount"`
}

type WithdrawResponse struct {
	Time             string  `json:"time"`
	AttemptNumber    int     `json:"attempt_number"`
	RecipientAddress string  `json:"recipient_address"`
	SenderAddress    string  `json:"sender_address"`
	Amount           float64 `json:"amount"`
	Success          bool    `json:"success"`
	Message          string  `json:"message"`
	Action           string  `json:"action"`
	SponsorUsed      bool    `json:"sponsor_used"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var writeMu sync.Mutex

func (s *Server) Withdraw(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.JSON(500, gin.H{"message": "Failed to upgrade to WebSocket"})
		return
	}

	var req WithdrawRequest
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.WriteJSON(gin.H{"message": "Invalid request"})
		return
	}

	err = json.Unmarshal(message, &req)
	if err != nil {
		conn.WriteJSON(gin.H{"message": "Malformed JSON"})
		return
	}

	kp, err := util.GetKeyFromSeed(req.SeedPhrase)
	if err != nil {
		s.sendErrorResponse(conn, "Invalid seed phrase")
		return
	}

	// Setup sponsor if provided
	var sponsor *wallet.SponsorWallet
	if req.SponsorSeedPhrase != "" {
		sponsor, err = wallet.NewSponsorWallet(req.SponsorSeedPhrase, s.wallet)
		if err != nil {
			s.sendErrorResponse(conn, "Invalid sponsor seed phrase")
			return
		}
	}

	// Immediate withdrawal of available balance
	s.withdrawAvailableBalance(conn, kp, req.WithdrawalAddress)

	// Schedule concurrent operations for locked balance
	s.scheduleConcurrentWithdraw(conn, kp, sponsor, req)
}

func (s *Server) withdrawAvailableBalance(conn *websocket.Conn, kp *keypair.Full, address string) {
	availableBalance, err := s.wallet.GetAvailableBalance(kp)
	if err != nil {
		s.sendResponse(conn, WithdrawResponse{
			Action:  "withdrawn",
			Message: "Error getting available balance: " + err.Error(),
			Success: false,
		})
		return
	}

	competitiveFee := util.GetCompetitiveFee(9400000, false) // Base 9.4 PI fee
	err = s.wallet.TransferWithFee(kp, availableBalance, address, competitiveFee)
	
	if err == nil {
		s.sendResponse(conn, WithdrawResponse{
			Action:  "withdrawn",
			Message: "Successfully withdrawn available balance",
			Success: true,
		})
	} else {
		s.sendResponse(conn, WithdrawResponse{
			Action:  "withdrawn",
			Message: "Error withdrawing available balance: " + err.Error(),
			Success: false,
		})
	}
}

func (s *Server) scheduleConcurrentWithdraw(conn *websocket.Conn, kp *keypair.Full, sponsor *wallet.SponsorWallet, req WithdrawRequest) {
	balance, err := s.wallet.GetClaimableBalance(req.LockedBalanceID)
	if err != nil {
		s.sendErrorResponse(conn, "Error getting claimable balance: "+err.Error())
		return
	}

	var unlockTime time.Time
	for _, claimant := range balance.Claimants {
		if claimant.Destination == kp.Address() {
			claimableAt, ok := util.ExtractClaimableTime(claimant.Predicate)
			if !ok {
				s.sendErrorResponse(conn, "Error finding locked balance unlock date")
				return
			}
			unlockTime = claimableAt
			break
		}
	}

	if unlockTime.IsZero() {
		s.sendErrorResponse(conn, "No valid claimant found for this wallet")
		return
	}

	s.sendResponse(conn, WithdrawResponse{
		Action:  "schedule",
		Message: fmt.Sprintf("Scheduled concurrent operations for %s", unlockTime.Format(time.RFC3339)),
		Success: true,
	})

	// Execute concurrent operations
	cfg := config.LoadConfig()
	processor := wallet.NewConcurrentProcessor(s.wallet, sponsor, cfg)
	
	ctx := context.Background()
	err = processor.ExecuteConcurrentOperations(
		ctx,
		kp,
		req.LockedBalanceID,
		req.WithdrawalAddress,
		unlockTime,
	)

	if err != nil {
		s.sendResponse(conn, WithdrawResponse{
			Action:      "completed",
			Message:     "Concurrent operations completed with some errors: " + err.Error(),
			Success:     false,
			SponsorUsed: sponsor != nil,
		})
	} else {
		s.sendResponse(conn, WithdrawResponse{
			Action:      "completed",
			Message:     "All concurrent operations completed successfully",
			Success:     true,
			SponsorUsed: sponsor != nil,
		})
	}
}

func (s *Server) sendResponse(conn *websocket.Conn, response WithdrawResponse) {
	writeMu.Lock()
	defer writeMu.Unlock()
	
	response.Time = time.Now().Format(time.RFC3339)
	conn.WriteJSON(response)
}

func (s *Server) sendErrorResponse(conn *websocket.Conn, message string) {
	s.sendResponse(conn, WithdrawResponse{
		Message: message,
		Success: false,
		Time:    time.Now().Format(time.RFC3339),
	})
}