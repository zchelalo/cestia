package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
)

func (s *Server) groupOptions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")
	cursor := r.URL.Query().Get("cursor")
	options, next, err := s.groups.GroupOptions(r.Context(), currentUser(r.Context()).ID, search, cursor, limit)
	if errors.Is(err, data.ErrInvalidOptionCursor) {
		s.writeError(w, r, apperr.BadRequest("invalid_cursor", "El cursor no es válido."))
		return
	}
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	s.writeJSON(w, r, http.StatusOK, options, meta(limit, next))
}

func (s *Server) memberOptions(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")
	cursor := r.URL.Query().Get("cursor")
	options, next, err := s.groups.MemberOptions(r.Context(), currentUser(r.Context()).ID, groupID, search, cursor, limit)
	if errors.Is(err, data.ErrInvalidOptionCursor) {
		s.writeError(w, r, apperr.BadRequest("invalid_cursor", "El cursor no es válido."))
		return
	}
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	s.writeJSON(w, r, http.StatusOK, options, meta(limit, next))
}

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.groups.List(r.Context(), currentUser(r.Context()).ID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, groups, nil)
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	group, err := s.groups.Create(r.Context(), currentUser(r.Context()).ID, body.Name, body.Description)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusCreated, group, nil)
}

func (s *Server) getGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	group, err := s.groups.Get(r.Context(), currentUser(r.Context()).ID, groupID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, group, nil)
}

func (s *Server) updateGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		ExpectedVersion int64  `json:"expected_version"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	group, err := s.groups.Update(r.Context(), currentUser(r.Context()).ID, groupID, body.Name, body.Description, body.ExpectedVersion)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, group, nil)
}

func (s *Server) archiveGroup(w http.ResponseWriter, r *http.Request) {
	s.setGroupArchived(w, r, true)
}

func (s *Server) restoreGroup(w http.ResponseWriter, r *http.Request) {
	s.setGroupArchived(w, r, false)
}

func (s *Server) setGroupArchived(w http.ResponseWriter, r *http.Request, archived bool) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body struct {
		ExpectedVersion int64 `json:"expected_version"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	group, err := s.groups.Archive(r.Context(), currentUser(r.Context()).ID, groupID, archived, body.ExpectedVersion)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, group, nil)
}

func (s *Server) members(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	members, err := s.groups.Members(r.Context(), currentUser(r.Context()).ID, groupID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, members, nil)
}

func (s *Server) setMemberRole(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	userID, err := parseUUID(r.PathValue("userId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.groups.SetRole(r.Context(), currentUser(r.Context()).ID, groupID, userID, body.Role); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	userID, err := parseUUID(r.PathValue("userId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.groups.RemoveMember(r.Context(), currentUser(r.Context()).ID, groupID, userID); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) leaveGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.groups.Leave(r.Context(), currentUser(r.Context()).ID, groupID); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) transferOwner(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body struct {
		UserID uuid.UUID `json:"user_id"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	group, err := s.groups.TransferOwner(r.Context(), currentUser(r.Context()).ID, groupID, body.UserID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, group, nil)
}

func (s *Server) invitations(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	items, err := s.groups.Invitations(r.Context(), currentUser(r.Context()).ID, groupID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, items, nil)
}

func (s *Server) invite(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	invitation, err := s.groups.Invite(r.Context(), currentUser(r.Context()).ID, groupID, body.Email)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	invitation.RawToken = ""
	s.writeJSON(w, r, http.StatusCreated, invitation, nil)
}

func (s *Server) resendInvitation(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	invitationID, err := parseUUID(r.PathValue("invitationId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body struct {
		ExpectedVersion int64 `json:"expected_version"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	invitation, err := s.groups.ResendInvitation(r.Context(), currentUser(r.Context()).ID, groupID, invitationID, body.ExpectedVersion)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	invitation.RawToken = ""
	s.writeJSON(w, r, http.StatusOK, invitation, nil)
}

func (s *Server) cancelInvitation(w http.ResponseWriter, r *http.Request) {
	groupID, err := parseUUID(r.PathValue("groupId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	invitationID, err := parseUUID(r.PathValue("invitationId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.groups.CancelInvitation(r.Context(), currentUser(r.Context()).ID, groupID, invitationID); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) previewInvitation(w http.ResponseWriter, r *http.Request) {
	var body tokenRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	invitation, err := s.groups.PreviewInvitation(r.Context(), currentUser(r.Context()), body.Token)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, invitation, nil)
}

func (s *Server) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	s.resolveInvitation(w, r, true)
}

func (s *Server) rejectInvitation(w http.ResponseWriter, r *http.Request) {
	s.resolveInvitation(w, r, false)
}

func (s *Server) resolveInvitation(w http.ResponseWriter, r *http.Request, accept bool) {
	var body tokenRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	invitation, err := s.groups.ResolveInvitation(r.Context(), currentUser(r.Context()), body.Token, accept)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, invitation, nil)
}
