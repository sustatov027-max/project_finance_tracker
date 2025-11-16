package models

type ByCategory struct {
	CategoryId        int     `json:"category_id"`
	CategoryName      string  `json:"category_name"`
	TotalAmount       float64 `json:"total_amount"`
	TransactionsCount int     `json:"transactions_count"`
	Percentage        float64 `json:"percentage"`
}
