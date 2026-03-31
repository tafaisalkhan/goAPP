package country

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
	items      []Country
	total      int64
	item       Country
	lastPatch  UpdatePatch
	lastLimit  int
	lastOffset int
}

func (s *stubRepo) List(ctx context.Context, limit, offset int) ([]Country, int64, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	return s.items, s.total, nil
}

func (s *stubRepo) Get(ctx context.Context, id int64) (Country, error) {
	return s.item, nil
}

func (s *stubRepo) Create(ctx context.Context, req Country) (Country, error) {
	return s.item, nil
}

func (s *stubRepo) Update(ctx context.Context, id int64, req UpdatePatch) (Country, error) {
	s.lastPatch = req
	return s.item, nil
}

func (s *stubRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

func TestHandlerListReturnsPaginatedCountries(t *testing.T) {
	h := NewHandler(&stubRepo{
		items: []Country{
			{ID: 1, CountryCode: "USA", CountryName: "United States", Code: "US"},
		},
		total: 25,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/countries?page=2&limit=10", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if len(got.Countries) != 1 {
		t.Fatalf("countries = %d, want 1", len(got.Countries))
	}

	if got.Pagination.Page != 2 || got.Pagination.Limit != 10 || got.Pagination.Total != 25 || got.Pagination.TotalPages != 3 {
		t.Fatalf("pagination = %#v", got.Pagination)
	}
}

func TestHandlerCreateReturnsCreatedCountry(t *testing.T) {
	h := NewHandler(&stubRepo{
		item: Country{ID: 1, CountryCode: "USA", CountryName: "United States", Code: "US"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/countries", strings.NewReader(`{"countrycode":"USA","countryname":"United States","code":"US"}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestHandlerUpdateAcceptsPartialPatch(t *testing.T) {
	repo := &stubRepo{
		item: Country{ID: 1, CountryCode: "USA", CountryName: "United States", Code: "US"},
	}
	h := NewHandler(repo)

	req := httptest.NewRequest(http.MethodPut, "/api/countries/1", strings.NewReader(`{"countryname":"United States of America"}`))
	rec := httptest.NewRecorder()
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if repo.lastPatch["countryname"] != "United States of America" {
		t.Fatalf("patch = %#v", repo.lastPatch)
	}
}
