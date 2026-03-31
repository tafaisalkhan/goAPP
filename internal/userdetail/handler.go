package userdetail

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	items, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load user details")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	item, err := h.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "user detail not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load user detail")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	var req UserDetail
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := validateRequest(req.FirstName, req.LastName, req.Email, req.Phone); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := h.repo.Create(r.Context(), normalizeDetail(req))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user detail")
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
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

	item, err := h.repo.Update(r.Context(), id, patch)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "user detail not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update user detail")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		writeError(w, http.StatusInternalServerError, "database connection is not configured")
		return
	}

	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "user detail not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete user detail")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		if !isUserDetailColumn(key) {
			return nil, errors.New("unknown field: " + key)
		}
		patch[key] = strings.TrimSpace(value)
	}

	return patch, nil
}

func validateRequest(firstName, lastName, email, phone string) error {
	if strings.TrimSpace(firstName) == "" {
		return errors.New("first_name is required")
	}
	if strings.TrimSpace(lastName) == "" {
		return errors.New("last_name is required")
	}
	if strings.TrimSpace(email) == "" {
		return errors.New("email is required")
	}
	if strings.TrimSpace(phone) == "" {
		return errors.New("phone is required")
	}

	return nil
}

func normalizeDetail(req UserDetail) UserDetail {
	req.UserID = strings.TrimSpace(req.UserID)
	req.UserName = strings.TrimSpace(req.UserName)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(req.Email)
	req.Enabled = strings.TrimSpace(req.Enabled)
	req.Role = strings.TrimSpace(req.Role)
	req.Password = strings.TrimSpace(req.Password)
	req.ParentID = strings.TrimSpace(req.ParentID)
	req.Secret = strings.TrimSpace(req.Secret)
	req.MFAEnabled = strings.TrimSpace(req.MFAEnabled)
	req.Type = strings.TrimSpace(req.Type)
	req.IsStaffMember = strings.TrimSpace(req.IsStaffMember)
	req.Organization = strings.TrimSpace(req.Organization)
	req.DOB = strings.TrimSpace(req.DOB)
	req.Gender = strings.TrimSpace(req.Gender)
	req.Phone = strings.TrimSpace(req.Phone)
	req.OpenStackUserID = strings.TrimSpace(req.OpenStackUserID)
	req.Status = strings.TrimSpace(req.Status)
	req.Active = strings.TrimSpace(req.Active)
	req.Image = strings.TrimSpace(req.Image)
	req.UpdatedBy = strings.TrimSpace(req.UpdatedBy)
	req.CreatedAt = strings.TrimSpace(req.CreatedAt)
	req.UpdatedAt = strings.TrimSpace(req.UpdatedAt)
	req.HystaxID = strings.TrimSpace(req.HystaxID)
	req.RoleName = strings.TrimSpace(req.RoleName)
	req.OpenStackDefaultProjectID = strings.TrimSpace(req.OpenStackDefaultProjectID)
	req.OpenStackID = strings.TrimSpace(req.OpenStackID)
	req.OtpEnabled = strings.TrimSpace(req.OtpEnabled)
	req.Description = strings.TrimSpace(req.Description)
	req.StripeID = strings.TrimSpace(req.StripeID)
	req.CardDetail = strings.TrimSpace(req.CardDetail)
	req.Address = strings.TrimSpace(req.Address)
	req.AllowCredit = strings.TrimSpace(req.AllowCredit)
	req.City = strings.TrimSpace(req.City)
	req.CodeExpiration = strings.TrimSpace(req.CodeExpiration)
	req.Country = strings.TrimSpace(req.Country)
	req.AgreedTermsVersion = strings.TrimSpace(req.AgreedTermsVersion)
	req.EmailVerified = strings.TrimSpace(req.EmailVerified)
	req.IsEnforced = strings.TrimSpace(req.IsEnforced)
	req.State = strings.TrimSpace(req.State)
	req.VerificationCode = strings.TrimSpace(req.VerificationCode)
	req.VerifyPhone = strings.TrimSpace(req.VerifyPhone)
	req.AllowFreeCredit = strings.TrimSpace(req.AllowFreeCredit)
	req.CreditLimitDate = strings.TrimSpace(req.CreditLimitDate)
	req.CardExemption = strings.TrimSpace(req.CardExemption)
	req.CurrentStep = strings.TrimSpace(req.CurrentStep)
	req.CardExcemption = strings.TrimSpace(req.CardExcemption)
	req.CreaditLimitDate = strings.TrimSpace(req.CreaditLimitDate)
	req.CurrentAgreedTermsVersion = strings.TrimSpace(req.CurrentAgreedTermsVersion)
	req.EnforceEnabled = strings.TrimSpace(req.EnforceEnabled)
	req.Firstname = strings.TrimSpace(req.Firstname)
	req.Lastname = strings.TrimSpace(req.Lastname)
	req.Test = strings.TrimSpace(req.Test)
	req.Username = strings.TrimSpace(req.Username)
	req.IsPasswordReset = strings.TrimSpace(req.IsPasswordReset)
	req.EmiratesID = strings.TrimSpace(req.EmiratesID)
	req.Fax = strings.TrimSpace(req.Fax)
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.Pobox = strings.TrimSpace(req.Pobox)
	req.TenantID = strings.TrimSpace(req.TenantID)
	req.NTNNo = strings.TrimSpace(req.NTNNo)
	return req
}

func isUserDetailColumn(name string) bool {
	for _, column := range userDetailColumns {
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
