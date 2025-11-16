package res

import(
	"encoding/json"
	"net/http"
)

func Json(w http.ResponseWriter, data any, statusCode int) error{
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}