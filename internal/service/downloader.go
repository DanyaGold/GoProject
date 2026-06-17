package service

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"go-project/internal/model"
	"go-project/internal/repository"
)

type DownloaderService struct {
	repo *repository.PostgresRepository
}

func NewDownloaderService(repo *repository.PostgresRepository) *DownloaderService {
	return &DownloaderService{repo: repo}
}

func (s *DownloaderService) StartDownload() error {
	go func() {
		prodURLs := []string{
			"https://api.dynamica.space/sources/source1.php",
			"https://api.dynamica.space/sources/source2.php",
			"https://api.dynamica.space/sources/source3.php",
		}
		clientURL := "https://api.dynamica.space/sources/clients.php"

		var wg sync.WaitGroup
		var mu sync.Mutex
		client := &http.Client{Timeout: 10 * time.Second}
		var allProducts []model.ExtProduct
		var allClients []model.ExtClient

		for _, url := range prodURLs {
			wg.Add(1)
			go func(u string) {
				defer wg.Done()
				var temp []model.ExtProduct
				req, _ := http.NewRequestWithContext(context.Background(), "GET", u, nil)
				if resp, err := client.Do(req); err == nil {
					defer resp.Body.Close()
					if err := json.NewDecoder(resp.Body).Decode(&temp); err == nil {
						mu.Lock()
						allProducts = append(allProducts, temp...)
						mu.Unlock()
					}
				}
			}(url)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			var temp []model.ExtClient
			req, _ := http.NewRequestWithContext(context.Background(), "GET", clientURL, nil)
			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()
				if err := json.NewDecoder(resp.Body).Decode(&temp); err == nil {
					mu.Lock()
					allClients = append(allClients, temp...)
					mu.Unlock()
				}
			}
		}()

		wg.Wait()

		log.Printf("Скачано продуктов: %d, клиентов: %d", len(allProducts), len(allClients))

		for i, p := range allProducts {
			allProducts[i].Price = s.cleanPrice(p.Price)
		}

		// Сохраняем продукты в базу
		if err := s.repo.SaveProducts(allProducts); err != nil {
			log.Printf("Ошибка сохранения продуктов: %v", err)
			return
		}

		// Сохраняем клиентов в базу
		if err := s.repo.SaveClients(allClients); err != nil {
			log.Printf("Ошибка сохранения клиентов: %v", err)
			return
		}

		log.Println("Данные успешно обновлены и сохранены в БД!")
	}()

	return nil
}

func (s *DownloaderService) cleanPrice(priceStr string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(priceStr, "")
}
