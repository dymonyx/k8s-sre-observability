package httpx

import (
	"encoding/json"
	"net/http"

	"chainwise/internal/model"
)

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, model.ErrorResponse{Error: message})
}
