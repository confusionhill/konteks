package service

// Pricing Parameters: Base Operations ($0.01 per 1,000 requests) + Tokens Processed ($0.07 per million)
const (
	PriceOperation     = 0.00001
	PriceProcessedPerM = 0.07000
)

// CalculateCost computes total monetary cost in USD based on Base Operation + Tokens Processed
func CalculateCost(tokensProcessed, tokensDelivered int) (costIn, costOut, costTotal float64) {
	costIn = (float64(tokensProcessed) / 1000000.0) * PriceProcessedPerM
	costOut = 0.0 // Free / do not bill output tokens
	costTotal = PriceOperation + costIn + costOut
	return
}
