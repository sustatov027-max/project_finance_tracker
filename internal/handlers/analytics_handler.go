package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"finance_tracker/internal/db"
	"finance_tracker/internal/models"
	"finance_tracker/internal/utils"
	"finance_tracker/pkg/res"
)



type AnalyticsHandler struct{}

func NewAnalyticsHandler(router *http.ServeMux) {
	handler := &AnalyticsHandler{}
	router.Handle("/api/analytics/summary", handler.SummaryHandler())
	router.Handle("/api/analytics/by_category", handler.ByCategoryHandler())
	router.Handle("/api/analytics/by_month", handler.ByMonthHandler())
}

func (handler *AnalyticsHandler) SummaryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userID = r.Context().Value(models.UserIDKey).(int)
		switch r.Method {
		case http.MethodGet:
			paramsRequest := r.URL.Query()
			dateStr := paramsRequest.Get("date")
			period := paramsRequest.Get("period")
			baseQuery := `SELECT 
    		COALESCE(SUM(CASE WHEN c.type = 'income' THEN t.amount END), 0) AS total_income,
    		COALESCE(SUM(CASE WHEN c.type = 'expense' THEN t.amount END), 0) AS total_expenses,
    		COALESCE(SUM(CASE WHEN c.type = 'income' THEN t.amount
                      		WHEN c.type = 'expense' THEN -t.amount END), 0) AS balance,
    		COUNT(t.id) AS transactions_count
			FROM transactions t
			JOIN categories c ON t.category_id = c.id`

			var startDate, endDate time.Time
			var finalQuery string

			if dateStr != "" || period != "" {
				var date time.Time
				if s := paramsRequest.Get("period"); s != "" {
					period = s
				} else {
					period = "day"
				}

				layout := "2006-01-02"
				dateStr := paramsRequest.Get("date")
				if dateStr == "" {
					dateStr = time.Now().Format(layout)
				}
				date, err := time.Parse(layout, dateStr)
				if err != nil {
					http.Error(w, "Invalid date", http.StatusBadRequest)
					return
				}

				startDate, endDate = utils.GetPeriodRange(period, date)
				finalQuery = baseQuery + ` WHERE t.user_id = $1 AND t.date >= $2 AND t.date < $3;`
			} else {
				finalQuery = baseQuery + `WHERE t.user_id = $1;`
			}

			var summary models.Summary

			var err error
			if dateStr != "" || period != "" {
				err = db.DB.QueryRow(finalQuery, userID, startDate, endDate).Scan(
					&summary.TotalIncome,
					&summary.TotalExpenses,
					&summary.Balance,
					&summary.TransactionsCount,
				)
			} else {
				err = db.DB.QueryRow(finalQuery, userID).Scan(
					&summary.TotalIncome,
					&summary.TotalExpenses,
					&summary.Balance,
					&summary.TransactionsCount,
				)
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if period == "" {
				period = "day"
			}
			summary.Period = period

			err = res.Json(w, summary, 200)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func (handler *AnalyticsHandler) ByCategoryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userID = r.Context().Value(models.UserIDKey).(int)
		switch r.Method {
		case http.MethodGet:
			paramsRequest := r.URL.Query()
			startDateStr := paramsRequest.Get("start_date")
			endDateStr := paramsRequest.Get("end_date")
			transactionType := paramsRequest.Get("type")

			layout := "2006-01-02"
			startDate, err := time.Parse(layout, startDateStr)
			if err != nil {
				http.Error(w, "Invalid start_date", http.StatusBadRequest)
				return
			}

			endDate, err := time.Parse(layout, endDateStr)
			if err != nil {
				http.Error(w, "Invalid end_date", http.StatusBadRequest)
				return
			}

			baseQuery := `SELECT 
					c.id AS category_id, 
					c.name AS category_name, 
					SUM(t.amount) AS total_amount,
					COUNT(t.id) AS transactions_count,
					(SUM(t.amount) / SUM(SUM(t.amount)) OVER()) * 100 AS percentage
					FROM transactions t
					JOIN categories c ON t.category_id = c.id
					WHERE t.user_id = $1 AND t.date >= $2 AND t.date <= $3`

			var rows *sql.Rows
			if transactionType != "" {
				finalQuery := baseQuery + ` AND c.type = $4 GROUP BY c.id, c.name;`

				rows, err = db.DB.Query(finalQuery, userID, startDate, endDate, transactionType)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				finalQuery := baseQuery + ` GROUP BY c.id, c.name;`

				rows, err = db.DB.Query(finalQuery, userID, startDate, endDate)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			defer rows.Close()
			var byCategories []models.ByCategory

			for rows.Next() {
				var c models.ByCategory
				err := rows.Scan(&c.CategoryId, &c.CategoryName, &c.TotalAmount, &c.TransactionsCount, &c.Percentage)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				byCategories = append(byCategories, c)
			}
			if err := rows.Err(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := res.Json(w, byCategories, 200); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func (handler *AnalyticsHandler) ByMonthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userID = r.Context().Value(models.UserIDKey).(int)
		switch r.Method {
		case http.MethodGet:
			paramsRequest := r.URL.Query()

			startDateStr := paramsRequest.Get("start_date")
			endDateStr := paramsRequest.Get("end_date")

			layout := "2006-01-02"
			startDate, err := time.Parse(layout, startDateStr)
			if err != nil {
				http.Error(w, "Invalid start_date", http.StatusBadRequest)
				return
			}

			endDate, err := time.Parse(layout, endDateStr)
			if err != nil {
				http.Error(w, "Invalid end_date", http.StatusBadRequest)
				return
			}

			query := `SELECT
					TO_CHAR(t.date, 'YYYY-MM') AS month,
					COALESCE(SUM(CASE WHEN c.type = 'income' THEN t.amount END), 0) AS total_income,
					COALESCE(SUM(CASE WHEN c.type = 'expense' THEN t.amount END), 0) AS total_expenses,
					COALESCE(SUM(CASE WHEN c.type = 'income' THEN t.amount 
								 WHEN c.type = 'expense' THEN -t.amount END), 0) AS balance,
					COUNT(t.id) AS transactions_count
					FROM transactions t
					JOIN categories c ON t.category_id = c.id
					WHERE t.user_id = $1 AND t.date >= $2 AND t.date <= $3
					GROUP BY TO_CHAR(t.date, 'YYYY-MM')
					ORDER BY month;`

			rows, err := db.DB.Query(query, userID, startDate, endDate)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var byMonths []models.ByMonth
			for rows.Next() {
				var month models.ByMonth

				err := rows.Scan(&month.Month, &month.TotalIncome, &month.TotalExpenses, &month.Balance, &month.TransactionsCount)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				byMonths = append(byMonths, month)
			}
			if err := rows.Err(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := res.Json(w, byMonths, 200); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}
