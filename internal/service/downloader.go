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
	repo       *repository.PostgresRepository
	cancelFunc context.CancelFunc
	mu         sync.Mutex
}

func NewDownloaderService(repo *repository.PostgresRepository) *DownloaderService {
	return &DownloaderService{repo: repo}
}

func (s *DownloaderService) StartDownload() error {
	s.mu.Lock()

	if s.cancelFunc != nil {
		log.Println("Останавливаем предыдущую активную задачу...")
		s.cancelFunc()
	}

	// создаем новый контекст с возможностью отмены
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel
	s.mu.Unlock()

	// завершаем таски в БД перед стартом
	_ = s.repo.CancelRunningTasks()

	taskID, err := s.repo.CreateTask()
	if err != nil {
		log.Printf("Не удалось создать таск в БД: %v", err)
		return err
	}

	go func(taskCtx context.Context, tID int) {
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

		// для продуктов
		for _, url := range prodURLs {
			wg.Add(1)
			go func(u string) {
				defer wg.Done()
				var temp []model.ExtProduct
				req, _ := http.NewRequestWithContext(taskCtx, "GET", u, nil)
				if resp, err := client.Do(req); err == nil {
					defer resp.Body.Close()
					if err := json.NewDecoder(resp.Body).Decode(&temp); err == nil {
						mu.Lock()
						allProducts = append(allProducts, temp...)
						mu.Unlock()
					}
				} else {
					log.Printf("Ошибка при загрузке данных с %s: %v", u, err)
				}
			}(url)
		}

		// для клиентов
		wg.Add(1)
		go func() {
			defer wg.Done()
			var temp []model.ExtClient
			req, _ := http.NewRequestWithContext(taskCtx, "GET", clientURL, nil)
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

		// выходим, если прерван новым POST запросом
		if taskCtx.Err() != nil {
			log.Printf("Задача №%d была прервана новой задачей. Выходим.", tID)
			return
		}

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

		_ = s.repo.CompleteTask(tID)
		log.Println("Данные успешно обновлены и сохранены в БД!")
	}(ctx, taskID)

	return nil
}

func (s *DownloaderService) cleanPrice(priceStr string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(priceStr, "")
}
