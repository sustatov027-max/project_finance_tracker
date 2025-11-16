package utils

import "strings"

func GetTransactionIdFromURL(path string) string{
	if strings.HasPrefix(path, "/api/transactions/") {
		idPart := strings.TrimPrefix(path, "/api/transactions/")

		parts := strings.Split(idPart, "/")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}
	return ""
}