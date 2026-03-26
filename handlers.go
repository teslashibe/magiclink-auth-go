package magiclink

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxJSONBodyBytes = 1 << 20 // 1 MiB

// HandleSend is a net/http handler for POST /auth/magic-link.
func (s *Service) HandleSend(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)

	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if err := s.Send(r.Context(), req.Email); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// HandleVerifyCode is a net/http handler for POST /auth/verify.
func (s *Service) HandleVerifyCode(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := s.VerifyCode(r.Context(), req.Email, req.Code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandleVerifyLink is a net/http handler for GET /auth/verify?token=....
func (s *Service) HandleVerifyLink(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	html, err := s.VerifyTokenPage(r.Context(), token)
	if err != nil {
		status := HTTPStatus(err)
		if status >= 500 {
			http.Error(w, PublicError(err), status)
			return
		}
		http.Error(w, PublicError(err), status)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

func decodeJSON(r *http.Request, out any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values are not allowed")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeError(w http.ResponseWriter, err error) {
	status := HTTPStatus(err)
	writeJSONError(w, status, PublicError(err))
}
