package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/modules/shopping"
)

type listRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	GroupID     *uuid.UUID `json:"group_id"`
}

type itemRequest struct {
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	Quantity           string     `json:"quantity"`
	Unit               string     `json:"unit"`
	EstimatedUnitPrice string     `json:"estimated_unit_price"`
	AssignedTo         *uuid.UUID `json:"assigned_to"`
	Notes              string     `json:"notes"`
	ExpectedVersion    int64      `json:"expected_version"`
}

type purchaseRequest struct {
	ExpectedVersion int64      `json:"expected_version"`
	ActualUnitPrice string     `json:"actual_unit_price"`
	PurchasedBy     uuid.UUID  `json:"purchased_by"`
	PaidBy          *uuid.UUID `json:"paid_by"`
	PurchasedAt     time.Time  `json:"purchased_at"`
	Notes           string     `json:"notes"`
}

type correctionRequest struct {
	ExpectedVersion int64      `json:"expected_version"`
	ActualUnitPrice string     `json:"actual_unit_price"`
	PurchasedBy     uuid.UUID  `json:"purchased_by"`
	PaidBy          *uuid.UUID `json:"paid_by"`
	PurchasedAt     time.Time  `json:"purchased_at"`
	Notes           string     `json:"notes"`
	Reason          string     `json:"reason"`
}

type reverseRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	Reason          string `json:"reason"`
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	userID := currentUser(r.Context()).ID
	groups, err := s.groups.List(r.Context(), userID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	lists, err := s.shopping.Lists(r.Context(), userID, nil, false)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	summary, err := s.shopping.Summary(r.Context(), userID, shopping.ExpenseFilter{IncludeArchived: true})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	purchases, _, err := s.shopping.PurchaseHistory(r.Context(), userID, shopping.HistoryFilter{Limit: 5})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	events, err := s.shopping.Activity(r.Context(), userID, nil, 8)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, map[string]any{
		"groups": groups, "personal_lists": lists, "summary": summary,
		"recent_purchases": purchases, "recent_activity": events,
	}, nil)
}

func (s *Server) listShoppingLists(w http.ResponseWriter, r *http.Request) {
	groupID, err := optionalUUID(r.URL.Query().Get("group_id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	includeArchived, _ := strconv.ParseBool(defaultString(r.URL.Query().Get("include_archived"), "false"))
	lists, err := s.shopping.Lists(r.Context(), currentUser(r.Context()).ID, groupID, includeArchived)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, lists, nil)
}

func (s *Server) createShoppingList(w http.ResponseWriter, r *http.Request) {
	var body listRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	list, err := s.shopping.CreateList(r.Context(), currentUser(r.Context()).ID, shopping.ListInput{
		Name: body.Name, Description: body.Description, GroupID: body.GroupID,
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusCreated, list, nil)
}

func (s *Server) getShoppingList(w http.ResponseWriter, r *http.Request) {
	listID, err := parseUUID(r.PathValue("listId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	list, err := s.shopping.GetList(r.Context(), currentUser(r.Context()).ID, listID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, list, nil)
}

func (s *Server) updateShoppingList(w http.ResponseWriter, r *http.Request) {
	listID, err := parseUUID(r.PathValue("listId"))
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
	list, err := s.shopping.UpdateList(r.Context(), currentUser(r.Context()).ID, listID,
		body.Name, body.Description, body.ExpectedVersion)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, list, nil)
}

func (s *Server) archiveShoppingList(w http.ResponseWriter, r *http.Request) {
	s.setListArchived(w, r, true)
}

func (s *Server) restoreShoppingList(w http.ResponseWriter, r *http.Request) {
	s.setListArchived(w, r, false)
}

func (s *Server) setListArchived(w http.ResponseWriter, r *http.Request, archived bool) {
	listID, err := parseUUID(r.PathValue("listId"))
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
	list, err := s.shopping.ArchiveList(r.Context(), currentUser(r.Context()).ID, listID, archived, body.ExpectedVersion)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, list, nil)
}

func (s *Server) listItems(w http.ResponseWriter, r *http.Request) {
	listID, err := parseUUID(r.PathValue("listId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	items, err := s.shopping.Items(r.Context(), currentUser(r.Context()).ID, listID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, items, nil)
}

func (s *Server) createItem(w http.ResponseWriter, r *http.Request) {
	listID, err := parseUUID(r.PathValue("listId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body itemRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	item, err := s.shopping.CreateItem(r.Context(), currentUser(r.Context()).ID, listID, itemInput(body))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusCreated, item, nil)
}

func (s *Server) updateItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := parseUUID(r.PathValue("itemId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body itemRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	item, err := s.shopping.UpdateItem(r.Context(), currentUser(r.Context()).ID, itemID, itemInput(body))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, item, nil)
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := parseUUID(r.PathValue("itemId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("expected_version"), 10, 64)
	if err != nil {
		s.writeError(w, r, apperr.BadRequest("invalid_version", "expected_version es obligatorio."))
		return
	}
	if err := s.shopping.DeleteItem(r.Context(), currentUser(r.Context()).ID, itemID, version); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) recordPurchase(w http.ResponseWriter, r *http.Request) {
	itemID, err := parseUUID(r.PathValue("itemId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body purchaseRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	purchase, err := s.shopping.RecordPurchase(r.Context(), currentUser(r.Context()).ID, itemID, shopping.PurchaseInput{
		ExpectedVersion: body.ExpectedVersion, ActualUnitPrice: body.ActualUnitPrice,
		PurchasedBy: body.PurchasedBy, PaidBy: body.PaidBy, PurchasedAt: body.PurchasedAt,
		Notes: body.Notes, IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusCreated, purchase, nil)
}

func (s *Server) getPurchase(w http.ResponseWriter, r *http.Request) {
	purchaseID, err := parseUUID(r.PathValue("purchaseId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	purchase, err := s.shopping.Purchase(r.Context(), currentUser(r.Context()).ID, purchaseID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, purchase, nil)
}

func (s *Server) correctPurchase(w http.ResponseWriter, r *http.Request) {
	purchaseID, err := parseUUID(r.PathValue("purchaseId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body correctionRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	purchase, err := s.shopping.CorrectPurchase(r.Context(), currentUser(r.Context()).ID, purchaseID, shopping.CorrectionInput{
		ExpectedVersion: body.ExpectedVersion, ActualUnitPrice: body.ActualUnitPrice,
		PurchasedBy: body.PurchasedBy, PaidBy: body.PaidBy, PurchasedAt: body.PurchasedAt,
		Notes: body.Notes, Reason: body.Reason, IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, purchase, nil)
}

func (s *Server) reversePurchase(w http.ResponseWriter, r *http.Request) {
	purchaseID, err := parseUUID(r.PathValue("purchaseId"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var body reverseRequest
	if err := decodeJSON(w, r, &body); err != nil {
		s.writeError(w, r, err)
		return
	}
	purchase, err := s.shopping.ReversePurchase(r.Context(), currentUser(r.Context()).ID, purchaseID, shopping.ReverseInput{
		ExpectedVersion: body.ExpectedVersion, Reason: body.Reason,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, purchase, nil)
}

func (s *Server) expensesSummary(w http.ResponseWriter, r *http.Request) {
	filter, err := expenseFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	summary, err := s.shopping.Summary(r.Context(), currentUser(r.Context()).ID, filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, summary, nil)
}

func (s *Server) expensesByList(w http.ResponseWriter, r *http.Request) {
	filter, err := expenseFilter(r)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	items, err := s.shopping.ExpensesByList(r.Context(), currentUser(r.Context()).ID, filter)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, items, nil)
}

func (s *Server) purchaseHistory(w http.ResponseWriter, r *http.Request) {
	groupID, err := optionalUUID(r.URL.Query().Get("group_id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	from, err := optionalTime(r.URL.Query().Get("from"))
	if err != nil {
		s.writeError(w, r, apperr.BadRequest("invalid_from", "La fecha desde no es válida."))
		return
	}
	to, err := optionalTime(r.URL.Query().Get("to"))
	if err != nil {
		s.writeError(w, r, apperr.BadRequest("invalid_to", "La fecha hasta no es válida."))
		return
	}
	listID, err := optionalUUID(r.URL.Query().Get("list_id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	paidBy, err := optionalUUID(r.URL.Query().Get("paid_by"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	purchasedBy, err := optionalUUID(r.URL.Query().Get("purchased_by"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	purchases, next, err := s.shopping.PurchaseHistory(r.Context(), currentUser(r.Context()).ID, shopping.HistoryFilter{
		GroupID: groupID, From: from, To: to, ListID: listID, PaidBy: paidBy,
		PurchasedBy: purchasedBy, Cursor: r.URL.Query().Get("cursor"), Limit: limit,
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	s.writeJSON(w, r, http.StatusOK, purchases, meta(limit, next))
}

func (s *Server) activity(w http.ResponseWriter, r *http.Request) {
	groupID, err := optionalUUID(r.URL.Query().Get("group_id"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := s.shopping.Activity(r.Context(), currentUser(r.Context()).ID, groupID, limit)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, events, nil)
}

func expenseFilter(r *http.Request) (shopping.ExpenseFilter, error) {
	groupID, err := optionalUUID(r.URL.Query().Get("group_id"))
	if err != nil {
		return shopping.ExpenseFilter{}, err
	}
	from, err := optionalTime(r.URL.Query().Get("from"))
	if err != nil {
		return shopping.ExpenseFilter{}, apperr.BadRequest("invalid_from", "La fecha desde no es válida.")
	}
	to, err := optionalTime(r.URL.Query().Get("to"))
	if err != nil {
		return shopping.ExpenseFilter{}, apperr.BadRequest("invalid_to", "La fecha hasta no es válida.")
	}
	includeArchived, err := strconv.ParseBool(defaultString(r.URL.Query().Get("include_archived"), "true"))
	if err != nil {
		return shopping.ExpenseFilter{}, apperr.BadRequest("invalid_include_archived", "include_archived no es válido.")
	}
	return shopping.ExpenseFilter{GroupID: groupID, From: from, To: to, IncludeArchived: includeArchived}, nil
}

func itemInput(body itemRequest) shopping.ItemInput {
	return shopping.ItemInput{
		Name: body.Name, Description: body.Description, Quantity: body.Quantity, Unit: body.Unit,
		EstimatedUnitPrice: body.EstimatedUnitPrice, AssignedTo: body.AssignedTo,
		Notes: body.Notes, Version: body.ExpectedVersion,
	}
}

func optionalUUID(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, apperr.BadRequest("invalid_id", "El identificador no es válido.")
	}
	return &id, nil
}

func optionalTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
