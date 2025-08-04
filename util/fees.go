package util

import (
	"math/rand"
	"time"
)

// GetCompetitiveFee returns a competitive fee to outperform other bots
func GetCompetitiveFee(baseAmount int64, isUrgent bool) int64 {
	rand.Seed(time.Now().UnixNano())
	
	if isUrgent {
		// For critical claiming operations, use maximum competitive fees
		return baseAmount + int64(rand.Intn(5000000)) // Add 0.5 PI randomness
	}
	
	// For regular operations
	return baseAmount + int64(rand.Intn(1000000)) // Add 0.1 PI randomness
}

// CalculateOptimalTiming returns the optimal time to start operations
func CalculateOptimalTiming(unlockTime time.Time) time.Time {
	// Start 100ms before unlock to beat competitors
	return unlockTime.Add(-100 * time.Millisecond)
}