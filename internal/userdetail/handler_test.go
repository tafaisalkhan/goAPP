package userdetail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type stubRepo struct {
	items     []UserDetail
	item      UserDetail
	lastPatch UpdatePatch
}

func (s *stubRepo) List(ctx context.Context) ([]UserDetail, error) {
	return s.items, nil
}

func (s *stubRepo) Get(ctx context.Context, id int64) (UserDetail, error) {
	return s.item, nil
}

func (s *stubRepo) Create(ctx context.Context, req UserDetail) (UserDetail, error) {
	return s.item, nil
}

func (s *stubRepo) Update(ctx context.Context, id int64, req UpdatePatch) (UserDetail, error) {
	s.lastPatch = req
	return s.item, nil
}

func (s *stubRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

func TestHandlerCreateReturnsCreatedUserDetail(t *testing.T) {
	h := NewHandler(&stubRepo{
		item: UserDetail{ID: 1, FirstName: "A", LastName: "B", Email: "a@b.com", Phone: "123"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user-details", strings.NewReader(`{"first_name":"A","last_name":"B","email":"a@b.com","phone":"123"}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestHandlerListReturnsArray(t *testing.T) {
	h := NewHandler(&stubRepo{
		items: []UserDetail{
			{ID: 1, FirstName: "A", LastName: "B", Email: "a@b.com", Phone: "123"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users-detail", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got []UserDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not a JSON array: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("items = %d, want 1", len(got))
	}

	if got[0].FirstName != "A" || got[0].Email != "a@b.com" {
		t.Fatalf("item = %#v", got[0])
	}
}

func TestHandlerUpdateAcceptsPartialPatch(t *testing.T) {
	repo := &stubRepo{
		item: UserDetail{ID: 1, FirstName: "Updated", LastName: "B", Email: "a@b.com", Phone: "123"},
	}
	h := NewHandler(repo)

	req := httptest.NewRequest(http.MethodPut, "/api/user-details/1", strings.NewReader(`{"first_name":"Updated"}`))
	rec := httptest.NewRecorder()
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if repo.lastPatch["first_name"] != "Updated" {
		t.Fatalf("patch = %#v, want first_name=Updated", repo.lastPatch)
	}
	if len(repo.lastPatch) != 1 {
		t.Fatalf("patch length = %d, want 1", len(repo.lastPatch))
	}
}
