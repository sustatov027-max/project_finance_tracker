package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"finance_tracker/internal/db"
	"finance_tracker/internal/models"
	"finance_tracker/internal/utils"
	"finance_tracker/pkg/res"
)

type TransactionHandler struct{}

func NewTransactionHandler(router *http.ServeMux) {
	handler := &TransactionHandler{}
	router.Handle("/api/transactions/", handler.Transaction())
}

func (handler *TransactionHandler) Transaction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		userID := r.Context().Value(models.UserIDKey).(int)

		switch r.Method {
		case http.MethodGet:
			id := utils.GetTransactionIdFromURL(r.URL.Path)
			paramsRequest := r.URL.Query()
			if id == "" {
				baseQuery := `SELECT id, amount, description, date, category_id, user_id, created_at FROM transactions WHERE user_id = $1`

				var conditions []string
				var args []interface{}
				args = append(args, userID)
				if s := paramsRequest.Get("category_id"); s != "" {
					id, err := strconv.Atoi(s)
					if err != nil {
						http.Error(w, "Invalid category_id", http.StatusBadRequest)
						return
					}
					args = append(args, id)
					conditions = append(conditions, fmt.Sprintf("category_id = $%d", len(args)))
				}
				layout := "2006-01-02"
				if s := paramsRequest.Get("start_date"); s != "" {
					startDate, err := time.Parse(layout, s)
					if err != nil {
						http.Error(w, "Invalid start_date", http.StatusBadRequest)
						return
					}
					args = append(args, startDate)
					conditions = append(conditions, fmt.Sprintf("date >= $%d", len(args)))
				}

				if s := paramsRequest.Get("end_date"); s != "" {
					endDate, err := time.Parse(layout, s)
					if err != nil {
						http.Error(w, "Invalid end_date", http.StatusBadRequest)
						return
					}
					args = append(args, endDate)
					conditions = append(conditions, fmt.Sprintf("date <= $%d", len(args)))
				}

				if s := paramsRequest.Get("min_amount"); s != "" {
					minAmount, err := strconv.Atoi(s)
					if err != nil {
						http.Error(w, "Invalid min_amount", http.StatusBadRequest)
						return
					}
					args = append(args, minAmount)
					conditions = append(conditions, fmt.Sprintf("amount >= $%d", len(args)))
				}

				if s := paramsRequest.Get("max_amount"); s != "" {
					maxAmount, err := strconv.Atoi(s)
					if err != nil {
						http.Error(w, "Invalid max_amount", http.StatusBadRequest)
						return
					}
					args = append(args, maxAmount)
					conditions = append(conditions, fmt.Sprintf("amount <= $%d", len(args)))
				}
				var finalQuery string
				if len(conditions) > 0 {
					finalQuery = baseQuery + " AND " + strings.Join(conditions, " AND ")
				} else {
					finalQuery = baseQuery
				}

				finalQuery += " ORDER BY id"
				fmt.Println(finalQuery)
				fmt.Println(len(conditions))

				rows, err := db.DB.Query(finalQuery, args...)
				if err != nil {
					http.Error(w, "Failed query", http.StatusInternalServerError)
					return
				}
				defer rows.Close()
				var transactions []models.Transaction

				for rows.Next() {
					var t models.Transaction
					err := rows.Scan(&t.Id, &t.Amount, &t.Description, &t.Date, &t.Category_id, &t.User_id, &t.Created_at)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					transactions = append(transactions, t)
				}
				if err := rows.Err(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				
				if err := res.Json(w, transactions, 200); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			} else {
				row := db.DB.QueryRow(`SELECT id, amount, description, date, category_id, user_id, created_at FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
				var t models.Transaction

				err := row.Scan(&t.Id, &t.Amount, &t.Description, &t.Date, &t.Category_id, &t.User_id, &t.Created_at)
				if err != nil {
					if err == sql.ErrNoRows {
						http.Error(w, "Transaction not found", http.StatusNotFound)
					} else {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
					return
				}

				if err := res.Json(w, t, 200); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		case http.MethodPost:
			var requestBody struct {
				Amount      float64 `json:"amount"`
				Description string  `json:"description"`
				CategoryId  int     `json:"category_id"`
				Date        string  `json:"date"`
			}

			w.Header().Set("Content-Type", "application/json")

			err := json.NewDecoder(r.Body).Decode(&requestBody)
			if err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if requestBody.Amount < 0 {
				http.Error(w, "Amount must be more null", http.StatusBadRequest)
				return
			}

			if requestBody.Description == "" {
				http.Error(w, "Description cannot be empty", http.StatusBadRequest)
				return
			}

			var categoryExists bool
			err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1 AND user_ID = $2)", requestBody.CategoryId, userID).Scan(&categoryExists)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !categoryExists {
				http.Error(w, "Category doesnt exist", http.StatusBadRequest)
				return
			}

			if requestBody.Date == "" {
				requestBody.Date = time.Now().Format("2006-01-02")
			}

			var createdTransaction models.Transaction
			err = db.DB.QueryRow(`INSERT INTO transactions (amount, description, date, category_id, user_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, amount, description, date, category_id, user_id, created_at`, requestBody.Amount, requestBody.Description, requestBody.Date, requestBody.CategoryId, userID).Scan(&createdTransaction.Id, &createdTransaction.Amount, &createdTransaction.Description, &createdTransaction.Date, &createdTransaction.Category_id, &createdTransaction.User_id, &createdTransaction.Created_at)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := res.Json(w, createdTransaction, 201); err != nil {
				http.Error(w, "Error encoding json", http.StatusInternalServerError)
				return
			}

		case http.MethodPut:
			var requestBody struct {
				Amount      *float64 `json:"amount"`
				Description *string  `json:"description"`
				CategoryId  *int     `json:"category_id"`
				Date        *string  `json:"date"`
			}

			id := utils.GetTransactionIdFromURL(r.URL.Path)

			if id == "" {
				http.Error(w, "Id cannot be empty", http.StatusBadRequest)
				return
			} else {
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				if err != nil {
					http.Error(w, "Invalid JSON", http.StatusBadRequest)
					return
				}

				var transactionExists bool
				err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM transactions WHERE id = $1 AND user_id = $2)", id, userID).Scan(&transactionExists)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if !transactionExists {
					http.Error(w, "Transaction doesnt exist", http.StatusBadRequest)
					return
				}

				if requestBody.CategoryId != nil {
					var categoryExists bool
					err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1 and user_id = $2)", requestBody.CategoryId, userID).Scan(&categoryExists)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if !categoryExists {
						http.Error(w, "Category doesnt exist", http.StatusBadRequest)
						return
					}
				}

				if requestBody.Amount != nil && *requestBody.Amount <= 0 {
					http.Error(w, "Amount must been more than null", http.StatusBadRequest)
					return
				}

				var updatedTransaction models.Transaction
				err = db.DB.QueryRow(`UPDATE transactions 
					SET amount = COALESCE($1, amount),
    				description = COALESCE($2, description), 
    				date = COALESCE($3, date),
    				category_id = COALESCE($4, category_id)
					WHERE id = $5 AND user_id = $6
					RETURNING id, amount, description, date, category_id, user_id, created_at`, requestBody.Amount, requestBody.Description, requestBody.Date, requestBody.CategoryId, id, userID).Scan(&updatedTransaction.Id, &updatedTransaction.Amount, &updatedTransaction.Description, &updatedTransaction.Date, &updatedTransaction.Category_id, &updatedTransaction.User_id, &updatedTransaction.Created_at)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := res.Json(w, updatedTransaction, 200); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		case http.MethodDelete:
			id := utils.GetTransactionIdFromURL(r.URL.Path)

			var transactionExists bool
			err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM transactions WHERE id = $1 AND user_id = $2)", id, userID).Scan(&transactionExists)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !transactionExists {
				http.Error(w, "Transaction doesnt exist", http.StatusNotFound)
				return
			}

			_, err = db.DB.Exec(`DELETE FROM transactions WHERE id = $1 AND user_id = $2;`, id, userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}
