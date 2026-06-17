package repository

import (
	"database/sql"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetStats() (map[string]int, error) {
	var products, clients, brands, categories int

	_ = r.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&products)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&clients)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM brands").Scan(&brands)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categories)

	stats := map[string]int{
		"products":   products,
		"clients":    clients,
		"brands":     brands,
		"categories": categories,
	}

	return stats, nil
}
