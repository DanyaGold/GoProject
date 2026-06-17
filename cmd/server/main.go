package main

import (
	"database/sql"
	"encoding/json"
	"go-project/internal/repository"
	"go-project/internal/service"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		// если запуск локальный
		dsn = "host=localhost port=5433 user=user password=1234 dbname=dynamica_db sslmode=disable"
	}

	// подключение к БД
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewPostgresRepository(db)
	svc := service.NewDownloaderService(repo)

	// возврат данных
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
			return
		}

		stats, err := repo.GetStats()
		if err != nil {
			http.Error(w, "Ошибка получения статистики", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// загрузка данных
	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = svc.StartDownload()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "Загрузка запущена"}`))
	})

	log.Println("Сервер запущен на порту :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
