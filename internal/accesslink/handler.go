package accesslink

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// RedeemPath is where the UI exchanges a link token for the API key. It sits
// outside /v1 so it needs no API key (the token is the credential).
const RedeemPath = "/access-link/redeem"

type redeemRequest struct {
	Token string `json:"token"`
}

type redeemResponse struct {
	APIKey string `json:"api_key"`
}

// RedeemHandler serves POST RedeemPath: {"token": "..."} -> {"api_key": "..."}.
// Invalid, expired, and reused links all get the same 401.
func (l *Links) RedeemHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var req redeemRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil || req.Token == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body must be {\"token\": \"...\"}"})
			return
		}

		key, err := l.Redeem(r.Context(), req.Token)
		if errors.Is(err, ErrInvalid) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": ErrInvalid.Error()})
			return
		}
		if err != nil {
			slog.ErrorContext(r.Context(), "access link redeem failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not redeem access link"})
			return
		}
		writeJSON(w, http.StatusOK, redeemResponse{APIKey: key})
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	// The response carries the API key: never cache it anywhere.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
