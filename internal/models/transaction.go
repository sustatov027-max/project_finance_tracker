package models

type Transaction struct {
	Id          int     `json:"id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
	Category_id int     `json:"category_id"`
	User_id     int     `json:"user_id"`
	Created_at  string  `json:"created_at"`
}
