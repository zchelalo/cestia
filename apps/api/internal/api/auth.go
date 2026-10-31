package api

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/modules/identity"
)

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenRequest struct {
	Token string `json:"token"`
}

type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type profileRequest struct {
	DisplayName string `json:"display_name"`
	Locale      string `json:"locale"`
	Timezone    string `json:"timezone"`
}

type deactivateRequest struct {
	Password string `json:"password"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var body registerRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	user, err := s.identity.Register(r.Context(), identity.RegisterInput{
		Email: body.Email, Password: body.Password, DisplayName: body.DisplayName, IP: clientIP(r),
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusCreated, userView(user), nil)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	result, err := s.identity.Login(r.Context(), body.Email, body.Password, clientIP(r))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.setSession(w, result.SessionToken, result.CSRFToken, result.ExpiresAt)
	s.writeJSON(w, r, http.StatusOK, map[string]any{
		"user": userView(result.User), "csrf_token": result.CSRFToken,
	}, nil)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	user, session := currentUser(r.Context()), currentSession(r.Context())
	if err := s.identity.Logout(r.Context(), user.ID, session.ID); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, r, http.StatusOK, userView(currentUser(r.Context())), nil)
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	var body profileRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	user, err := s.identity.UpdateProfile(r.Context(), currentUser(r.Context()).ID, body.DisplayName, body.Locale, body.Timezone)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, userView(user), nil)
}

func (s *Server) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var body tokenRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.identity.VerifyEmail(r.Context(), body.Token); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resendVerification(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	if err := s.identity.ResendVerification(r.Context(), user.ID, user.Email); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusAccepted, map[string]string{"message": "Si la cuenta requiere verificación, enviamos un correo."}, nil)
}

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.identity.ForgotPassword(r.Context(), body.Email, clientIP(r)); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusAccepted, map[string]string{
		"message": "Si existe una cuenta con ese correo, enviamos instrucciones para recuperar el acceso.",
	}, nil)
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body resetRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.identity.ResetPassword(r.Context(), body.Token, body.Password); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var body changePasswordRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	user := currentUser(r.Context())
	if err := s.identity.ChangePassword(r.Context(), user.ID, body.CurrentPassword, body.NewPassword); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deactivate(w http.ResponseWriter, r *http.Request) {
	var body deactivateRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	user := currentUser(r.Context())
	if err := s.identity.Deactivate(r.Context(), user.ID, body.Password); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	user, current := currentUser(r.Context()), currentSession(r.Context())
	items, err := s.identity.Sessions(r.Context(), user.ID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	now := time.Now().UTC()
	views := make([]map[string]any, 0, len(items))
	for _, session := range items {
		views = append(views, sessionView(session, session.ID == current.ID, now))
	}
	s.writeJSON(w, r, http.StatusOK, views, nil)
}

func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	sessionID, err := parseUUID(r.PathValue("sessionId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.identity.RevokeSession(r.Context(), user.ID, sessionID); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func parseUUID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperr.BadRequest("invalid_id", "El identificador no es válido.")
	}
	return id, nil
}
