package service

// DeepSeek V4 Flash Pricing Parameters
const (
	PriceInputPerM  = 0.0679 // $0.0679 per 1,000,000 input tokens
	PriceOutputPerM = 0.168  // $0.168 per 1,000,000 output tokens
)

// CalculateCost computes input cost, output cost, and total monetary cost in USD
func CalculateCost(tokensInput, tokensOutput int) (costIn, costOut, costTotal float64) {
	costIn = (float64(tokensInput) / 1000000.0) * PriceInputPerM
	costOut = (float64(tokensOutput) / 1000000.0) * PriceOutputPerM
	costTotal = costIn + costOut
	return
}
