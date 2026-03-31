package userdetail

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var ErrNotFound = errors.New("user detail not found")

type Repository interface {
	List(ctx context.Context) ([]UserDetail, error)
	Get(ctx context.Context, id int64) (UserDetail, error)
	Create(ctx context.Context, req UserDetail) (UserDetail, error)
	Update(ctx context.Context, id int64, patch UpdatePatch) (UserDetail, error)
	Delete(ctx context.Context, id int64) error
}

type MySQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) List(ctx context.Context) ([]UserDetail, error) {
	if err := r.requireDB(); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT id, %s FROM user_detail ORDER BY id", strings.Join(quotedColumns(), ", "))
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]UserDetail, 0)
	for rows.Next() {
		var item UserDetail
		if err := scanUserDetail(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *MySQLRepository) Get(ctx context.Context, id int64) (UserDetail, error) {
	if err := r.requireDB(); err != nil {
		return UserDetail{}, err
	}

	query := fmt.Sprintf("SELECT id, %s FROM user_detail WHERE id = ?", strings.Join(quotedColumns(), ", "))
	var item UserDetail
	if err := scanUserDetail(r.db.QueryRowContext(ctx, query, id), &item); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserDetail{}, ErrNotFound
		}
		return UserDetail{}, err
	}

	return item, nil
}

func (r *MySQLRepository) Create(ctx context.Context, req UserDetail) (UserDetail, error) {
	if err := r.requireDB(); err != nil {
		return UserDetail{}, err
	}

	columns := quotedColumns()
	placeholders := strings.TrimRight(strings.Repeat("?, ", len(columns)), ", ")
	query := fmt.Sprintf("INSERT INTO user_detail (%s) VALUES (%s)", strings.Join(columns, ", "), placeholders)

	result, err := r.db.ExecContext(ctx, query, valuesFromUserDetail(req, false)...)
	if err != nil {
		return UserDetail{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return UserDetail{}, err
	}

	return r.Get(ctx, id)
}

func (r *MySQLRepository) Update(ctx context.Context, id int64, patch UpdatePatch) (UserDetail, error) {
	if err := r.requireDB(); err != nil {
		return UserDetail{}, err
	}

	if len(patch) == 0 {
		return UserDetail{}, errors.New("update patch cannot be empty")
	}

	assignments := make([]string, 0, len(patch))
	args := make([]any, 0, len(patch)+1)
	for _, column := range userDetailColumns {
		value, ok := patch[column]
		if !ok {
			continue
		}

		assignments = append(assignments, quoteColumn(column)+" = ?")
		args = append(args, value)
	}

	if len(assignments) == 0 {
		return UserDetail{}, errors.New("no valid fields provided")
	}

	query := fmt.Sprintf("UPDATE user_detail SET %s WHERE id = ?", strings.Join(assignments, ", "))
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return UserDetail{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return UserDetail{}, err
	}
	if rowsAffected == 0 {
		return UserDetail{}, ErrNotFound
	}

	return r.Get(ctx, id)
}

func (r *MySQLRepository) Delete(ctx context.Context, id int64) error {
	if err := r.requireDB(); err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM user_detail WHERE id = ?`, id)
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
	cols := make([]string, len(userDetailColumns))
	for i, column := range userDetailColumns {
		cols[i] = quoteColumn(column)
	}
	return cols
}

func valuesFromUserDetail(detail UserDetail, includeID bool) []any {
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

func scanUserDetail(scanner rowScanner, detail *UserDetail) error {
	value := reflect.ValueOf(detail).Elem()
	dests := make([]any, 0, value.NumField())
	dests = append(dests, &detail.ID)

	for i := 1; i < value.NumField(); i++ {
		dests = append(dests, value.Field(i).Addr().Interface())
	}

	return scanner.Scan(dests...)
}
