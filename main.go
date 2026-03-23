package main

/*
PIPELINE:
Fetch Ad URLs from Pages -> Fetch Ad data -> Process with AI
*/

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/lmittmann/tint"
	"github.com/ollama/ollama/api"
)

var (
	client         = http.Client{}
	adCounter uint = 0 // sync this under a mutex
)

func main() {
	initConfig()
	initLogger()
	InitCache()

	urls := make(chan string)
	adUrls := make(chan string)
	adDatas := make(chan AdData)
	procAdDatas := make(chan AdData)

	cache := NewCache()
	cache.Load()

	ctx, cancel := context.WithCancel(context.Background())
	// It is always best to keep this at 1 worker, so that pages are processed sequentially
	processPages(ctx, urls, adUrls, cancel, 1)
	processAds(adUrls, adDatas, cache, int(cfg.Jobs))
	// AI processing is done with Ollama and as such always ends up being the bottleneck.
	// Thus it is best to keep it at 1 worker
	processAiData(adDatas, procAdDatas, cache, 1)

	go func() {
		defer close(urls)
		for p := range cfg.Pages {
			select {
			case <-ctx.Done():
				return
			default:
				urls <- fmt.Sprintf(
					"https://www.olx.uz/%s/?currency=UYE&page=%d",
					cfg.Category, p+1,
				)
			}
		}
	}()

	writeOutput(procAdDatas)

	fmt.Printf(`
Summary:
    Ads processed: %d`,
		adCounter)
}

func initLogger() {
	level := slog.LevelInfo
	if cfg.Verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			AddSource:  true,
			TimeFormat: time.DateTime,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Value.Kind() == slog.KindAny {
					if _, ok := a.Value.Any().(error); ok {
						return tint.Attr(9, a)
					}
				} else if a.Key == slog.MessageKey {
					return tint.Attr(4, a)
				}
				return a
			},
		}),
	))
}

func processPages(
	ctx context.Context,
	urls <-chan string,
	out chan<- string,
	cancel context.CancelFunc,
	workers int,
) {
	runWithWorkers(
		workers,
		func() {
			for url := range urls {
				select {
				case <-ctx.Done():
					return
				default:
					getAdUrls(out, cancel, url)
				}
			}
		},
		func() {
			close(out)
		},
	)
}

func processAds(urls <-chan string, out chan<- AdData, cache Cache, workers int) {
	runWithWorkers(
		workers,
		func() {
			for url := range urls {
				getAdData(out, url, cache)
			}
		},
		func() {
			close(out)
		},
	)
}

func processAiData(adDatas <-chan AdData, out chan<- AdData, cache Cache, workers int) {
	if !cfg.AiProcessing {
		go func() {
			for adData := range adDatas {
				out <- adData
			}
			close(out)
		}()
		return
	}

	runWithWorkers(
		workers,
		func() {
			for adData := range adDatas {
				getAiProcessedData(out, adData, cache)
			}
		},
		func() {
			close(out)
		},
	)
}

func writeOutput(datas <-chan AdData) {
	outputPath := "output.csv"
	f, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		slog.Error("failed to open file", "path", outputPath, "error", err)
		os.Exit(1)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	wroteHeader := false
	for data := range datas {
		if !wroteHeader {
			headers := data.CsvHeaders()
			slog.Debug("writing headers", "headers", headers)
			writer.Write(headers)
			wroteHeader = true
		}
		slog.Debug("saving ad data", "data", data)
		writer.Write(data.CsvRow())
	}
}

func getAdUrls(out chan<- string, cancel context.CancelFunc, url string) {
	contentReader, err := fetch(url)
	if err != nil {
		slog.Error("could not fetch", "url", url, "error", err)
		return
	}
	defer contentReader.Close()

	doc, err := goquery.NewDocumentFromReader(contentReader)
	if err != nil {
		slog.Error("could not create html parser", "error", err)
		return
	}

	doc.Find(`div[data-cy="l-card"]`).EachWithBreak(func(i int, s *goquery.Selection) bool {
		if cfg.MaxAds != 0 && adCounter >= cfg.MaxAds {
			cancel()
			return false
		}

		adRef, exists := s.Find("a").Attr("href")
		if !exists {
			slog.Warn("didn't find anchor with href attr")
			return true
		}
		out <- "https://www.olx.uz" + adRef
		adCounter++
		return true
	})
}

func getAdData(out chan<- AdData, url string, cache Cache) {
	if adData, exists := cache[url]; exists {
		adData.StructuredData = nil
		out <- adData
		return
	}

	contentReader, err := fetch(url)
	if err != nil {
		slog.Error("could not fetch", "url", url, "error", err)
		return
	}
	defer contentReader.Close()

	doc, err := goquery.NewDocumentFromReader(contentReader)
	if err != nil {
		slog.Error("could not create html parser", "error", err)
		return
	}

	adData := AdData{
		Id:        getId(doc),
		Date:      getDate(doc),
		Price:     getPrice(doc),
		Condition: getCondition(doc),
		Name:      getName(doc),
		Desc:      getDesc(doc),
		Url:       url,
	}

	cache[url] = adData
	if err := cache.Save(); err != nil {
		slog.Error("failed to save cache", "error", err)
		return
	}

	out <- adData
}

func getAiProcessedData(out chan<- AdData, adData AdData, cache Cache) {
	if adData, exists := cache[adData.Url]; exists && adData.StructuredData != nil {
		out <- adData
		return
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		slog.Error("failed to create ollama client", "error", err)
		return
	}

	req := &api.ChatRequest{
		Model: "gemma3n",
		Messages: []api.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Title: %s\n\n%s", adData.Name, adData.Desc)},
		},
		Stream: ptr(false),
	}

	err = client.Chat(context.Background(), req, func(resp api.ChatResponse) error {
		result := strings.TrimSpace(resp.Message.Content)
		result = strings.TrimPrefix(result, "```json")
		result = strings.TrimPrefix(result, "```")
		result = strings.TrimSuffix(result, "```")
		result = strings.TrimSpace(result)

		adData.StructuredData = NewOrderedMap[string, any]()
		if err := json.Unmarshal([]byte(result), adData.StructuredData); err != nil {
			return fmt.Errorf("failed to unmarshal to json: %w\n%s", err, resp.Message.Content)
		}

		cache[adData.Url] = adData
		if err := cache.Save(); err != nil {
			return fmt.Errorf("failed to save cache: %w", err)
		}
		return nil
	})
	if err != nil {
		slog.Error("failed to get ollama response", "error", err)
		return
	}
	out <- adData
}

func fetch(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	var htmlFilepath string
	if strings.HasPrefix(url, "https://www.olx.uz/d/obyavlenie/") {
		htmlFilepath = path.Join(
			getAdsDir(),
			fmt.Sprintf("ad_%s", strings.TrimPrefix(req.URL.Path, "/d/obyavlenie/")),
		)
	} else {
		htmlFilepath = path.Join(
			getPagesDir(),
			fmt.Sprintf("page_%s.html", req.URL.Query().Get("page")),
		)
	}

	f, err := os.Open(htmlFilepath)
	if err == nil {
		slog.Info("using cached", "url", url)
		return f, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("error opening file %s: %w", htmlFilepath, err)
	}

	slog.Info("fetching", "url", url)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}

	f, err = os.Create(htmlFilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %s: %w", htmlFilepath, err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to cache into file %s: %w", htmlFilepath, err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to start: %w", err)
	}
	return f, nil
}

func getId(doc *goquery.Document) uint {
	var id uint
	doc.Find(`div[data-cy="ad-footer-bar-section"] span`).EachWithBreak(
		func(i int, s *goquery.Selection) bool {
			text := s.Text()
			if strings.Contains(text, "ID") {
				parts := strings.SplitN(text, ":", 2)
				if len(parts) == 2 {
					idStr := strings.TrimSpace(parts[1])
					tempId, err := strconv.Atoi(idStr)
					if err != nil {
						slog.Error("couldn't parse id", "id", idStr, "error", err)
					} else {
						id = uint(tempId)
					}
					return false
				}
			}
			return true
		},
	)
	return id
}

func getDate(doc *goquery.Document) Date {
	var date Date
	text := doc.Find(`span[data-cy="ad-posted-at"]`).Text()
	dateStr := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(text)), "опубликовано ")
	if strings.HasPrefix(dateStr, "сегодня") {
		date = Date{Time: time.Now()}
	} else {
		parsedDate, err := parseRussianDate(dateStr)
		if err != nil {
			slog.Error("couldn't parse date", "date", dateStr, "error", err)
		} else {
			date = parsedDate
		}
	}
	return date
}

const exchangeRate = 12000 // $ -> UZS

func getPrice(doc *goquery.Document) float32 {
	text := doc.Find(`div[data-testid="ad-price-container"] h3`).Text()
	text = strings.ToLower(strings.ReplaceAll(text, " ", ""))
	if substr, cut := strings.CutSuffix(text, "у.е."); cut {
		price, err := strconv.Atoi(substr)
		if err != nil {
			slog.Error("failed to convert price string to uint", "str", substr, "error", err)
			return 0
		}
		return float32(price)
	} else if substr, cut := strings.CutSuffix(text, "сум"); cut {
		price, err := strconv.Atoi(substr)
		if err != nil {
			slog.Error("failed to convert price string to uint", "str", substr, "error", err)
			return 0
		}
		return float32(price) / exchangeRate
	}
	slog.Error("unknown price format", "str", text)
	return 0
}

func getCondition(doc *goquery.Document) Condition {
	var condition Condition
	doc.Find(`p[class="css-odhutu"]`).EachWithBreak(
		func(i int, s *goquery.Selection) bool {
			text := s.Text()
			if strings.HasPrefix(text, "Состояние:") {
				if strings.HasSuffix(text, "Новый") {
					condition = ConditionNew
				} else if strings.HasSuffix(text, "Б/у") {
					condition = ConditionUsed
				} else {
					slog.Error("couldn't identify condition", "condition", text)
					condition = ConditionUnknown
				}
				return false
			}
			return true
		},
	)
	return condition
}

func getName(doc *goquery.Document) string {
	return strings.TrimSpace(doc.Find(`div[data-cy="offer_title"] h4`).Text())
}

func getDesc(doc *goquery.Document) string {
	return strings.TrimSpace(doc.Find(`div[data-cy="ad_description"] div`).Text())
}

func runWithWorkers(workers int, f func(), cleanup func()) {
	var wg sync.WaitGroup
	for range workers {
		wg.Go(f)
	}
	go func() {
		wg.Wait()
		cleanup()
	}()
}

var russianMonths = map[string]string{
	"января":   "January",
	"февраля":  "February",
	"марта":    "March",
	"апреля":   "April",
	"мая":      "May",
	"июня":     "June",
	"июля":     "July",
	"августа":  "August",
	"сентября": "September",
	"октября":  "October",
	"ноября":   "November",
	"декабря":  "December",
}

func parseRussianDate(s string) (Date, error) {
	for ru, en := range russianMonths {
		s = strings.TrimSuffix(strings.ReplaceAll(s, ru, en), " г.")
	}
	dateTime, err := time.Parse("2 January 2006", s)
	return Date{Time: dateTime}, err
}

func ptr[T any](val T) *T {
	return &val
}

const systemPrompt = `
You are a structured data extractor for PC hardware listings.

Extract PC part information from the provided text and return it as JSON.
Normalize all values to be consistent across different listings.

Rules:
- Return ONLY valid JSON with no markdown, no backticks, no explanation.
- Normalize component names to their canonical form:
  - CPU: use format "Intel Core i7-13700K" or "AMD Ryzen 5 7600X"
  - GPU: use format "NVIDIA GeForce RTX 4070" or "AMD Radeon RX 7800 XT"
  - RAM: use format "16GB DDR5 5600MHz"
  - Storage: use format "1TB NVMe SSD" or "2TB HDD"
  - Motherboard: use format "ASUS ROG Strix B650-A"
- Normalize all storage sizes to GB or TB (e.g. "500GB", "1TB")
- Normalize all RAM sizes to GB (e.g. "16GB")
- Normalize all frequencies to MHz (e.g. "3200MHz")
- If a value is not mentioned, use null
- If a value is ambiguous or unclear, use null

Return this exact JSON structure:
{
  "cpu": null,
  "gpu": null,
  "ram": null,
  "storage": [],
  "motherboard": null,
  "cpu_cooler": null,
  "case": null,
  "psu": null,
  "os": null
}
`
