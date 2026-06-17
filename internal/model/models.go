package model

type ExtProduct struct {
	ID       int    `json:"id"`
	Stock    int    `json:"stock"`
	Name     string `json:"name"`
	Brand    string `json:"brand"`
	Category string `json:"category"`
	Price    string `json:"price"`
}

type ExtClient struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Products  []int  `json:"products"`
}
