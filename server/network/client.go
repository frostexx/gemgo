package network

import (
	"errors"
	"log"
	"time"
)

type PiClient struct {
	pool *ConnectionPool
}

func NewPiClient(networkURL string) (*PiClient, error) {
	pool, err := NewConnectionPool(networkURL, 10) // Pool of 10 connections
	if err != nil {
		return nil, err
	}
	return &PiClient{pool: pool}, nil
}

// SubmitTransaction is a placeholder for the actual transaction submission logic.
// In a real implementation, this would use the stellar/go library to build,
// sign, and submit the transaction to the network.
func (c *PiClient) SubmitTransaction(txType, mainSeed, sponsorSeed string, fee int64) error {
	conn := c.pool.Get()
	defer c.pool.Release(conn)

	log.Printf("Submitting %s tx with fee %d via connection %d", txType, fee, conn.id)

	// Simulate network latency and potential for failure
	time.Sleep(time.Duration(50+time.Now().UnixNano()%150) * time.Millisecond)

	// Simulate higher failure rate for lower fees
	if fee < 5000000 && time.Now().UnixNano()%2 == 0 {
		return errors.New("network rejected transaction: insufficient fee")
	}

	// Simulate random network errors
	if time.Now().UnixNano()%10 == 0 {
		return errors.New("random network error: connection timed out")
	}

	return nil // Placeholder for success
}