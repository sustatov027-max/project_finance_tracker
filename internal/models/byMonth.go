package models

type ByMonth struct {
	Month             string  `json:"month"`
	TotalIncome       float64 `json:"total_income"`
	TotalExpenses     float64 `json:"total_expenses"`
	Balance           float64 `json:"balance"`
	TransactionsCount int     `json:"transactions_count"`
}
