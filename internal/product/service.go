package product

import (
	"context"
	"database/sql"
	"errors"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	if s.db == nil {
		return nil, errors.New("database connection is not configured")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, price_cents
		FROM products
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceCents); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
