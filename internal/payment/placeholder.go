package payment

// PaymentResult 支付结果
type PaymentResult struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	URL       string `json:"url"`
}
