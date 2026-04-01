package country

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	page, limit, ok := parsePagination(w, r)
	if !ok {
		return
	}

	offset := (page - 1) * limit
	items, total, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load countries")
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	writeJSON(w, http.StatusOK, ListResponse{
		Countries: items,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "country not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load country")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	var req Country
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := validateRequest(req.CountryCode, req.CountryName); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.service.Create(r.Context(), normalizeCountry(req))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create country")
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	patch, err := decodePatch(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(patch) == 0 {
		writeError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	item, err := h.service.Update(r.Context(), id, patch)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "country not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update country")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "country not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete country")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parsePagination(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	page := 1
	limit := 10

	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid page")
			return 0, 0, false
		}
		page = value
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return 0, 0, false
		}
		limit = value
	}

	return page, limit, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}

	return true
}

func decodePatch(r *http.Request) (UpdatePatch, error) {
	var raw map[string]string
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, errors.New("invalid JSON body")
	}

	patch := make(UpdatePatch, len(raw))
	for key, value := range raw {
		if !isCountryColumn(key) {
			return nil, errors.New("unknown field: " + key)
		}
		patch[key] = strings.TrimSpace(value)
	}

	return patch, nil
}

func validateRequest(countryCode, countryName string) error {
	if strings.TrimSpace(countryCode) == "" {
		return errors.New("countrycode is required")
	}
	if strings.TrimSpace(countryName) == "" {
		return errors.New("countryname is required")
	}

	return nil
}

func normalizeCountry(req Country) Country {
	req.CountryCode = strings.TrimSpace(req.CountryCode)
	req.CountryName = strings.TrimSpace(req.CountryName)
	req.Code = strings.TrimSpace(req.Code)
	return req
}

func isCountryColumn(name string) bool {
	for _, column := range countryColumns {
		if column == name {
			return true
		}
	}

	return false
}

func parseID(w http.ResponseWriter, raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}

	return id, true
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}
