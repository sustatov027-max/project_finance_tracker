package utils

import (
	"time"
)

func GetPeriodRange(period string, date time.Time) (startDate, endDate time.Time) {
	switch period {
	case "day":
		startDate = date
		endDate = date.AddDate(0, 0, 1)
	case "week":
		weekday := int(date.Weekday())
		startDate = date.AddDate(0, 0, -weekday)
		endDate = startDate.AddDate(0, 0, 7)
	case "month":
		startDate = time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
		endDate = startDate.AddDate(0, 1, 0)
	case "year":
		startDate = time.Date(date.Year(), 1, 1, 0, 0, 0, 0, date.Location())
		endDate = startDate.AddDate(1, 0, 0)
	default:
		startDate = date.AddDate(0, 0, -1)
		endDate = date
	}
	return startDate, endDate

}
