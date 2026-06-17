package repository

import (
	"database/sql"
	"go-project/internal/model"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateTask() (int, error) {
	var id int
	err := r.db.QueryRow(`
		INSERT INTO tasks (status, started_at) 
		VALUES ('в работе', NOW()) 
		RETURNING id`).Scan(&id)
	return id, err
}

func (r *PostgresRepository) CompleteTask(id int) error {
	_, err := r.db.Exec(`
		UPDATE tasks 
		SET status = 'завершено', ended_at = NOW() 
		WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) CancelRunningTasks() error {
	_, err := r.db.Exec(`
		UPDATE tasks 
		SET status = 'завершено', ended_at = NOW() 
		WHERE status = 'в работе'`)
	return err
}

func (r *PostgresRepository) SaveClients(clients []model.ExtClient) error {
	for _, c := range clients {
		_, err := r.db.Exec(`
			INSERT INTO clients (id, first_name, last_name) VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE SET first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name`,
			c.ID, c.FirstName, c.LastName)
		if err != nil {
			return err
		}

		_, _ = r.db.Exec("DELETE FROM client_products WHERE client_id = $1", c.ID)

		for _, productID := range c.Products {
			_, _ = r.db.Exec(`
				INSERT INTO client_products (client_id, product_id) VALUES ($1, $2)
				ON CONFLICT DO NOTHING`, c.ID, productID)
		}
	}
	return nil
}

func (r *PostgresRepository) SaveProducts(products []model.ExtProduct) error {
	for _, p := range products {
		var brandID, categoryID int

		err := r.db.QueryRow(`
			INSERT INTO brands (name) VALUES ($1) 
			ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name 
			RETURNING id`, p.Brand).Scan(&brandID)
		if err != nil {
			return err
		}

		err = r.db.QueryRow(`
			INSERT INTO categories (name) VALUES ($1) 
			ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name 
			RETURNING id`, p.Category).Scan(&categoryID)
		if err != nil {
			return err
		}

		_, err = r.db.Exec(`
			INSERT INTO products (id, name, price, stock, brand_id, category_id) 
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET 
				name=EXCLUDED.name, 
				price=EXCLUDED.price, 
				stock=EXCLUDED.stock,
				brand_id=EXCLUDED.brand_id,
				category_id=EXCLUDED.category_id`,
			p.ID, p.Name, p.Price, p.Stock, brandID, categoryID)
		if err != nil {
			return err
		}
	}
	return nil
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
