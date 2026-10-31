package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/config"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/modules/groups"
	"github.com/zchelalo/cestia/apps/api/internal/modules/identity"
	"github.com/zchelalo/cestia/apps/api/internal/modules/shopping"
	"github.com/zchelalo/cestia/apps/api/internal/secure"
)

type Server struct {
	cfg      config.Config
	identity *identity.Service
	groups   *groups.Service
	shopping *shopping.Service
	logger   *slog.Logger
	mux      *http.ServeMux
}

type contextKey string

const (
	ctxRequestID contextKey = "request_id"
	ctxUser      contextKey = "user"
	ctxSession   contextKey = "session"
)

func New(cfg config.Config, identityService *identity.Service, groupService *groups.Service,
	shoppingService *shopping.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{cfg: cfg, identity: identityService, groups: groupService, shopping: shoppingService, logger: logger}
	s.mux = http.NewServeMux()
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.requestID(s.recoverer(s.originGuard(s.mux)))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	s.mux.HandleFunc("POST /api/v1/auth/register", s.register)
	s.mux.HandleFunc("POST /api/v1/auth/login", s.login)
	s.mux.HandleFunc("POST /api/v1/auth/verify-email", s.verifyEmail)
	s.mux.HandleFunc("POST /api/v1/auth/forgot-password", s.forgotPassword)
	s.mux.HandleFunc("POST /api/v1/auth/reset-password", s.resetPassword)

	s.mux.Handle("GET /api/v1/me", s.authenticated(http.HandlerFunc(s.me)))
	s.mux.Handle("PATCH /api/v1/me", s.authenticated(s.csrf(http.HandlerFunc(s.updateMe))))
	s.mux.Handle("POST /api/v1/me/deactivate", s.authenticated(s.csrf(http.HandlerFunc(s.deactivate))))
	s.mux.Handle("GET /api/v1/me/sessions", s.authenticated(http.HandlerFunc(s.sessions)))
	s.mux.Handle("DELETE /api/v1/me/sessions/{sessionId}", s.authenticated(s.csrf(http.HandlerFunc(s.revokeSession))))
	s.mux.Handle("POST /api/v1/auth/logout", s.authenticated(s.csrf(http.HandlerFunc(s.logout))))
	s.mux.Handle("POST /api/v1/auth/change-password", s.authenticated(s.csrf(http.HandlerFunc(s.changePassword))))
	s.mux.Handle("POST /api/v1/auth/resend-verification", s.authenticated(s.csrf(http.HandlerFunc(s.resendVerification))))

	s.mux.Handle("GET /api/v1/dashboard", s.private(http.HandlerFunc(s.dashboard)))
	s.mux.Handle("GET /api/v1/groups", s.private(http.HandlerFunc(s.listGroups)))
	s.mux.Handle("POST /api/v1/groups", s.private(s.csrf(http.HandlerFunc(s.createGroup))))
	s.mux.Handle("GET /api/v1/groups/{groupId}", s.private(http.HandlerFunc(s.getGroup)))
	s.mux.Handle("PATCH /api/v1/groups/{groupId}", s.private(s.csrf(http.HandlerFunc(s.updateGroup))))
	s.mux.Handle("POST /api/v1/groups/{groupId}/archive", s.private(s.csrf(http.HandlerFunc(s.archiveGroup))))
	s.mux.Handle("POST /api/v1/groups/{groupId}/restore", s.private(s.csrf(http.HandlerFunc(s.restoreGroup))))
	s.mux.Handle("POST /api/v1/groups/{groupId}/transfer-ownership", s.private(s.csrf(http.HandlerFunc(s.transferOwner))))
	s.mux.Handle("POST /api/v1/groups/{groupId}/leave", s.private(s.csrf(http.HandlerFunc(s.leaveGroup))))
	s.mux.Handle("GET /api/v1/groups/{groupId}/members", s.private(http.HandlerFunc(s.members)))
	s.mux.Handle("PATCH /api/v1/groups/{groupId}/members/{userId}", s.private(s.csrf(http.HandlerFunc(s.setMemberRole))))
	s.mux.Handle("DELETE /api/v1/groups/{groupId}/members/{userId}", s.private(s.csrf(http.HandlerFunc(s.removeMember))))
	s.mux.Handle("GET /api/v1/groups/{groupId}/invitations", s.private(http.HandlerFunc(s.invitations)))
	s.mux.Handle("POST /api/v1/groups/{groupId}/invitations", s.private(s.csrf(http.HandlerFunc(s.invite))))
	s.mux.Handle("POST /api/v1/groups/{groupId}/invitations/{invitationId}/resend", s.private(s.csrf(http.HandlerFunc(s.resendInvitation))))
	s.mux.Handle("DELETE /api/v1/groups/{groupId}/invitations/{invitationId}", s.private(s.csrf(http.HandlerFunc(s.cancelInvitation))))
	s.mux.Handle("POST /api/v1/invitations/preview", s.private(s.csrf(http.HandlerFunc(s.previewInvitation))))
	s.mux.Handle("POST /api/v1/invitations/accept", s.private(s.csrf(http.HandlerFunc(s.acceptInvitation))))
	s.mux.Handle("POST /api/v1/invitations/reject", s.private(s.csrf(http.HandlerFunc(s.rejectInvitation))))

	s.mux.Handle("GET /api/v1/shopping-lists", s.private(http.HandlerFunc(s.listShoppingLists)))
	s.mux.Handle("POST /api/v1/shopping-lists", s.private(s.csrf(http.HandlerFunc(s.createShoppingList))))
	s.mux.Handle("GET /api/v1/shopping-lists/{listId}", s.private(http.HandlerFunc(s.getShoppingList)))
	s.mux.Handle("PATCH /api/v1/shopping-lists/{listId}", s.private(s.csrf(http.HandlerFunc(s.updateShoppingList))))
	s.mux.Handle("POST /api/v1/shopping-lists/{listId}/archive", s.private(s.csrf(http.HandlerFunc(s.archiveShoppingList))))
	s.mux.Handle("POST /api/v1/shopping-lists/{listId}/restore", s.private(s.csrf(http.HandlerFunc(s.restoreShoppingList))))
	s.mux.Handle("GET /api/v1/shopping-lists/{listId}/items", s.private(http.HandlerFunc(s.listItems)))
	s.mux.Handle("POST /api/v1/shopping-lists/{listId}/items", s.private(s.csrf(http.HandlerFunc(s.createItem))))
	s.mux.Handle("PATCH /api/v1/items/{itemId}", s.private(s.csrf(http.HandlerFunc(s.updateItem))))
	s.mux.Handle("DELETE /api/v1/items/{itemId}", s.private(s.csrf(http.HandlerFunc(s.deleteItem))))
	s.mux.Handle("POST /api/v1/items/{itemId}/purchases", s.private(s.csrf(http.HandlerFunc(s.recordPurchase))))
	s.mux.Handle("GET /api/v1/purchases/{purchaseId}", s.private(http.HandlerFunc(s.getPurchase)))
	s.mux.Handle("PATCH /api/v1/purchases/{purchaseId}", s.private(s.csrf(http.HandlerFunc(s.correctPurchase))))
	s.mux.Handle("POST /api/v1/purchases/{purchaseId}/reverse", s.private(s.csrf(http.HandlerFunc(s.reversePurchase))))
	s.mux.Handle("GET /api/v1/expenses/summary", s.private(http.HandlerFunc(s.expensesSummary)))
	s.mux.Handle("GET /api/v1/expenses/by-list", s.private(http.HandlerFunc(s.expensesByList)))
	s.mux.Handle("GET /api/v1/purchases", s.private(http.HandlerFunc(s.purchaseHistory)))
	s.mux.Handle("GET /api/v1/activity", s.private(http.HandlerFunc(s.activity)))
}

func (s *Server) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie(s.sessionCookieName())
		if err != nil {
			s.writeError(w, r, apperr.Unauthorized("missing_session", "Inicia sesión para continuar."))
			return
		}
		user, session, err := s.identity.Authenticate(r.Context(), token.Value)
		if err != nil {
			s.clearSession(w)
			s.writeError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUser, user)
		ctx = context.WithValue(ctx, ctxSession, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) private(next http.Handler) http.Handler {
	return s.authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r.Context())
		if !user.Verified() {
			s.writeError(w, r, apperr.New(http.StatusForbidden, "email_unverified", "Verifica tu correo para continuar."))
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := currentSession(r.Context())
		provided := r.Header.Get("X-CSRF-Token")
		if provided == "" || !secureEqual(session.CSRFHash, provided) {
			s.writeError(w, r, apperr.Forbidden("La solicitud no pudo verificarse. Recarga la página."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && strings.TrimRight(origin, "/") != s.cfg.AppOrigin {
			s.writeError(w, r, apperr.Forbidden("Origen de solicitud no permitido."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.Must(uuid.NewV7()).String()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxRequestID, id)))
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("http panic", "stage", "panic", "request_id", requestID(r.Context()), "panic", recovered, "stack", string(debug.Stack()))
				s.writeError(w, r, apperr.Internal(errors.New("panic")))
			}
		}()
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds(), "request_id", requestID(r.Context()))
	})
}

func (s *Server) sessionCookieName() string {
	if s.cfg.Env == "production" {
		return "__Host-cestia_session"
	}
	return "cestia_session"
}

func (s *Server) csrfCookieName() string { return "cestia_csrf" }

func (s *Server) setSession(w http.ResponseWriter, token, csrf string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: s.sessionCookieName(), Value: token, Path: "/", HttpOnly: true,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: expires})
	http.SetCookie(w, &http.Cookie{Name: s.csrfCookieName(), Value: csrf, Path: "/", HttpOnly: false,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: expires})
}

func (s *Server) clearSession(w http.ResponseWriter) {
	expired := time.Unix(1, 0)
	http.SetCookie(w, &http.Cookie{Name: s.sessionCookieName(), Value: "", Path: "/", HttpOnly: true,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: expired, MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: s.csrfCookieName(), Value: "", Path: "/", HttpOnly: false,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: expired, MaxAge: -1})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return apperr.BadRequest("invalid_json", "El cuerpo de la solicitud es obligatorio.")
		}
		return apperr.BadRequest("invalid_json", "El cuerpo de la solicitud no es válido.")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return apperr.BadRequest("invalid_json", "El cuerpo debe contener un único objeto JSON.")
	}
	return nil
}

func currentUser(ctx context.Context) data.User {
	user, _ := ctx.Value(ctxUser).(data.User)
	return user
}

func currentSession(ctx context.Context) data.Session {
	session, _ := ctx.Value(ctxSession).(data.Session)
	return session
}

func requestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxRequestID).(string)
	return id
}

func secureEqual(hash []byte, raw string) bool {
	provided := secure.Hash(raw)
	if len(hash) != len(provided) {
		return false
	}
	var diff byte
	for i := range hash {
		diff |= hash[i] ^ provided[i]
	}
	return diff == 0
}
