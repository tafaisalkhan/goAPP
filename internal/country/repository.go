package country

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
)

var ErrNotFound = errors.New("country not found")

type Repository interface {
	List(ctx context.Context, limit, offset int) ([]Country, int64, error)
	Get(ctx context.Context, id int64) (Country, error)
	Create(ctx context.Context, req Country) (Country, error)
	Update(ctx context.Context, id int64, patch UpdatePatch) (Country, error)
	Delete(ctx context.Context, id int64) error
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) List(ctx context.Context, limit, offset int) ([]Country, int64, error) {
	if err := r.requireDB(); err != nil {
		return nil, 0, err
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM country`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf("SELECT id, %s FROM country ORDER BY id LIMIT ? OFFSET ?", strings.Join(quotedColumns(), ", "))
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Country, 0)
	for rows.Next() {
		var item Country
		if err := scanCountry(rows, &item); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *MySQLRepository) Get(ctx context.Context, id int64) (Country, error) {
	if err := r.requireDB(); err != nil {
		return Country{}, err
	}

	return getCountryByID(ctx, r.db, id)
}

func (r *MySQLRepository) Create(ctx context.Context, req Country) (Country, error) {
	if err := r.requireDB(); err != nil {
		return Country{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Country{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	columns := quotedColumns()
	placeholders := strings.TrimRight(strings.Repeat("?, ", len(columns)), ", ")
	query := fmt.Sprintf("INSERT INTO country (%s) VALUES (%s)", strings.Join(columns, ", "), placeholders)

	result, err := tx.ExecContext(ctx, query, valuesFromCountry(req, false)...)
	if err != nil {
		log.Printf("[DB_ERROR] operation=insert_country query=%s values=%v error=%v",
			query, req, err)
		return Country{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Country{}, err
	}

	item, err := getCountryByID(ctx, tx, id)
	if err != nil {
		return Country{}, err
	}

	if err := tx.Commit(); err != nil {
		return Country{}, err
	}

	return item, nil
}

func (r *MySQLRepository) Update(ctx context.Context, id int64, patch UpdatePatch) (Country, error) {
	if err := r.requireDB(); err != nil {
		return Country{}, err
	}

	if len(patch) == 0 {
		return Country{}, errors.New("update patch cannot be empty")
	}

	assignments := make([]string, 0, len(patch))
	args := make([]any, 0, len(patch)+1)
	for _, column := range countryColumns {
		value, ok := patch[column]
		if !ok {
			continue
		}

		assignments = append(assignments, quoteColumn(column)+" = ?")
		args = append(args, value)
	}

	if len(assignments) == 0 {
		return Country{}, errors.New("no valid fields provided")
	}

	query := fmt.Sprintf("UPDATE country SET %s WHERE id = ?", strings.Join(assignments, ", "))
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return Country{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Country{}, err
	}
	if rowsAffected == 0 {
		return Country{}, ErrNotFound
	}

	return r.Get(ctx, id)
}

func (r *MySQLRepository) Delete(ctx context.Context, id int64) error {
	if err := r.requireDB(); err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM country WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *MySQLRepository) requireDB() error {
	if r.db == nil {
		return errors.New("database connection is not configured")
	}

	return nil
}

func quotedColumns() []string {
	cols := make([]string, len(countryColumns))
	for i, column := range countryColumns {
		cols[i] = quoteColumn(column)
	}
	return cols
}

func valuesFromCountry(detail Country, includeID bool) []any {
	value := reflect.ValueOf(detail)
	start := 0
	if !includeID {
		start = 1
	}

	values := make([]any, 0, value.NumField()-start)
	for i := start; i < value.NumField(); i++ {
		values = append(values, value.Field(i).Interface())
	}

	return values
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCountry(scanner rowScanner, detail *Country) error {
	value := reflect.ValueOf(detail).Elem()
	dests := make([]any, 0, value.NumField())
	dests = append(dests, &detail.ID)

	for i := 1; i < value.NumField(); i++ {
		dests = append(dests, value.Field(i).Addr().Interface())
	}

	return scanner.Scan(dests...)
}

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func getCountryByID(ctx context.Context, querier rowQuerier, id int64) (Country, error) {
	query := fmt.Sprintf("SELECT id, %s FROM country WHERE id = ?", strings.Join(quotedColumns(), ", "))
	var item Country
	if err := scanCountry(querier.QueryRowContext(ctx, query, id), &item); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Country{}, ErrNotFound
		}
		return Country{}, err
	}

	return item, nil
}
