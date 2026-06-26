package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Toph4er/family-library/internal/models"
	"golang.org/x/time/rate"
)

// olConfig holds Open Library API configuration loaded from environment.
var olConfig = LoadOLConfig()

// --- Open Library search response types ---

// OLSearchResult represents a single book from the OL /search.json endpoint.
type OLSearchResult struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	AuthorName       []string `json:"author_name,omitempty"`
	FirstPublishYear *int     `json:"first_publish_year,omitempty"`
	CoverI           *int     `json:"cover_i,omitempty"`
	Subject          []string `json:"subject,omitempty"`
	Language         []string `json:"language,omitempty"`
	ISBN             []string `json:"isbn,omitempty"`
}

// OLSearchResponse is the top-level envelope from OL /search.json.
type OLSearchResponse struct {
	NumFound int              `json:"numFound"`
	Start    int              `json:"start"`
	Docs     []OLSearchResult `json:"docs"`
}

// buildLookupResponse builds the JSON response map from a looked-up Book.
func buildLookupResponse(book *models.Book, coverSource string) map[string]interface{} {
	resp := map[string]interface{}{
		"title":        book.Title,
		"cover_source": coverSource,
	}
	if book.Subtitle != nil {
		resp["subtitle"] = *book.Subtitle
	}
	if book.Authors != nil {
		resp["authors"] = *book.Authors
	}
	if book.Illustrators != nil {
		resp["illustrators"] = *book.Illustrators
	}
	if book.Publisher != nil {
		resp["publisher"] = *book.Publisher
	}
	if book.PublicationYear != nil {
		resp["publication_year"] = *book.PublicationYear
	}
	if book.PageCount != nil {
		resp["page_count"] = *book.PageCount
	}
	if book.BookType != nil {
		resp["book_type"] = *book.BookType
	}
	if book.ReadingLevels != nil {
		resp["reading_levels"] = *book.ReadingLevels
	}
	if book.Genres != nil {
		resp["genres"] = *book.Genres
	}
	if book.Themes != nil {
		resp["themes"] = *book.Themes
	}
	if book.Awards != nil {
		resp["awards"] = *book.Awards
	}
	if book.CoverImageURL != nil {
		resp["cover_image_url"] = *book.CoverImageURL
	}
	if book.DeweyDecimalClass != nil {
		resp["dewey_decimal_class"] = *book.DeweyDecimalClass
	}
	if book.Description != nil {
		resp["description"] = *book.Description
	}
	if book.Language != nil {
		resp["language"] = *book.Language
	}
	if book.SubjectPlaces != nil {
		resp["subject_places"] = *book.SubjectPlaces
	}
	if book.SubjectPeople != nil {
		resp["subject_people"] = *book.SubjectPeople
	}
	if book.SubjectTimes != nil {
		resp["subject_times"] = *book.SubjectTimes
	}
	if book.AgeRange != nil {
		resp["age_range"] = *book.AgeRange
	}
	if book.Series != nil {
		resp["series"] = *book.Series
	}
	return resp
}

// bookFromLookupResponse reconstructs a *models.Book from a lookup response map.
func bookFromLookupResponse(resp map[string]interface{}) *models.Book {
	book := &models.Book{Title: toString(resp["title"])}
	if v, ok := resp["subtitle"].(string); ok && v != "" {
		book.Subtitle = &v
	}
	if v, ok := resp["authors"].(string); ok && v != "" {
		book.Authors = &v
	}
	if v, ok := resp["illustrators"].(string); ok && v != "" {
		book.Illustrators = &v
	}
	if v, ok := resp["publisher"].(string); ok && v != "" {
		book.Publisher = &v
	}
	if v, ok := resp["publication_year"].(float64); ok {
		y := int(v)
		book.PublicationYear = &y
	}
	if v, ok := resp["page_count"].(float64); ok {
		p := int(v)
		book.PageCount = &p
	}
	if v, ok := resp["book_type"].(string); ok && v != "" {
		book.BookType = &v
	}
	if v, ok := resp["reading_levels"].(string); ok && v != "" {
		book.ReadingLevels = &v
	}
	if v, ok := resp["genres"].(string); ok && v != "" {
		book.Genres = &v
	}
	if v, ok := resp["themes"].(string); ok && v != "" {
		book.Themes = &v
	}
	if v, ok := resp["awards"].(string); ok && v != "" {
		book.Awards = &v
	}
	if v, ok := resp["cover_image_url"].(string); ok && v != "" {
		book.CoverImageURL = &v
	}
	if v, ok := resp["dewey_decimal_class"].(string); ok && v != "" {
		book.DeweyDecimalClass = &v
	}
	if v, ok := resp["description"].(string); ok && v != "" {
		book.Description = &v
	}
	if v, ok := resp["language"].(string); ok && v != "" {
		book.Language = &v
	}
	if v, ok := resp["subject_places"].(string); ok && v != "" {
		book.SubjectPlaces = &v
	}
	if v, ok := resp["subject_people"].(string); ok && v != "" {
		book.SubjectPeople = &v
	}
	if v, ok := resp["subject_times"].(string); ok && v != "" {
		book.SubjectTimes = &v
	}
	if v, ok := resp["age_range"].(string); ok && v != "" {
		book.AgeRange = &v
	}
	if v, ok := resp["series"].(string); ok && v != "" {
		book.Series = &v
	}
	return book
}

// cachedFetchFromOpenLibrary checks the SQLite cache first, then fetches from Open Library.
func cachedFetchFromOpenLibrary(db *sql.DB, isbn string, force bool) (*models.Book, string, error) {
	if !force {
		var cachedData, cachedAt string
		err := db.QueryRow("SELECT data, fetched_at FROM isbn_cache WHERE isbn = ?", isbn).Scan(&cachedData, &cachedAt)
		if err == nil {
			if t, parseErr := time.Parse(time.RFC3339, cachedAt); parseErr == nil && time.Since(t) < olConfig.CacheTTL {
				var resp map[string]interface{}
				if jsonErr := json.Unmarshal([]byte(cachedData), &resp); jsonErr == nil {
					book := bookFromLookupResponse(resp)
					if coverSource, ok := resp["cover_source"].(string); ok {
						return book, coverSource, nil
					}
					return book, "open_library", nil
				}
			}
		}
	}

	book, coverSource, err := fetchFromOpenLibrary(isbn)
	if err != nil || book == nil {
		return book, coverSource, err
	}

	resp := buildLookupResponse(book, coverSource)
	dataJSON, _ := json.Marshal(resp)
	_, _ = db.Exec(
		`INSERT OR REPLACE INTO isbn_cache (isbn, data, fetched_at) VALUES (?, ?, ?)`,
		isbn, string(dataJSON), time.Now().UTC().Format(time.RFC3339),
	)
	cacheHours := int(olConfig.CacheTTL.Hours())
	_, _ = db.Exec(
		`DELETE FROM isbn_cache WHERE datetime(fetched_at) < datetime('now', '-' || ? || ' hours')`,
		cacheHours,
	)

	return book, coverSource, nil
}

// apiHTTPClient is shared across all API fetches.
var apiHTTPClient = &http.Client{Timeout: olConfig.HTTPTimeout}

// olRateLimiter caps outgoing Open Library requests at 2 req/s (burst of 1).
var olRateLimiter = rate.NewLimiter(rate.Every(time.Second/time.Duration(olConfig.RateLimitPerSec)), 1)

func waitOLRateLimit(ctx context.Context) error {
	return olRateLimiter.Wait(ctx)
}

// olRequestWithRetry performs an HTTP GET with rate limiting and exponential backoff retry.
func olRequestWithRetry(ctx context.Context, url string, maxRetries int) ([]byte, int, error) {
	var body []byte
	var lastErr error
	var lastStatus int

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := waitOLRateLimit(ctx); err != nil {
			return nil, 0, fmt.Errorf("rate limiter wait: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) // #nosec G704 — OL API endpoint
		if err != nil {
			return nil, 0, fmt.Errorf("building Open Library request: %w", err)
		}
		req.Header.Set("User-Agent", olConfig.UserAgent)

		resp, err := apiHTTPClient.Do(req) // #nosec G704 — OL API endpoint
		if err != nil {
			lastErr = fmt.Errorf("Open Library HTTP request: %w", err)
			lastStatus = 0
			continue
		}

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading Open Library response: %w", err)
			lastStatus = resp.StatusCode
			continue
		}

		lastStatus = resp.StatusCode

		if resp.StatusCode == http.StatusOK {
			return body, resp.StatusCode, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if retryAfter > 0 {
				slog.Warn("Open Library rate limit hit, honouring Retry-After", "url", url, "retry_after", retryAfter.String())
				if err := sleepCtx(ctx, retryAfter); err != nil {
					return nil, resp.StatusCode, fmt.Errorf("context cancelled during 429 backoff: %w", err)
				}
			}
		}

		lastErr = fmt.Errorf("Open Library returned status %d", resp.StatusCode)
	}

	if lastStatus != http.StatusOK {
		backoff := exponentialBackoff(maxRetries, 500*time.Millisecond, 5*time.Second)
		slog.Info("Open Library retry backoff before final failure", "url", url, "backoff", backoff.String())
		_ = sleepCtx(context.Background(), backoff)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("Open Library request failed after %d attempts", maxRetries+1)
	}
	return body, lastStatus, lastErr
}

func parseRetryAfter(header string) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(header); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		if delay := time.Until(t); delay > 0 {
			return delay
		}
	}
	return 0
}

func exponentialBackoff(attempt int, base, maxDelay time.Duration) time.Duration {
	delay := base * (1 << uint(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// yearRe matches a 4-digit year at the start of a string.
var yearRe = regexp.MustCompile(`^(\d{4})`)

// fetchFromOpenLibrary fetches book metadata from the Open Library API.
func fetchFromOpenLibrary(isbn string) (*models.Book, string, error) {
	u := fmt.Sprintf("%s/isbn/%s.json", olConfig.BaseURL, url.PathEscape(isbn))
	body, statusCode, err := olRequestWithRetry(context.Background(), u, 2)
	if err != nil {
		return nil, "", fmt.Errorf("Open Library request failed after retries: %w", err)
	}

	if statusCode == http.StatusNotFound {
		return nil, "", nil
	}
	if statusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Open Library returned status %d", statusCode)
	}

	var olResp map[string]interface{}
	if err := json.Unmarshal(body, &olResp); err != nil {
		return nil, "", fmt.Errorf("parsing Open Library JSON: %w", err)
	}

	book := &models.Book{Title: toString(olResp["title"])}

	if subtitle, ok := olResp["subtitle"].(string); ok && subtitle != "" {
		book.Subtitle = &subtitle
	}

	if authorsRaw, ok := olResp["authors"].([]interface{}); ok && len(authorsRaw) > 0 {
		authors := resolveOpenLibraryAuthorKeys(authorsRaw)
		if len(authors) > 0 {
			authorsJSON, _ := json.Marshal(authors)
			s := string(authorsJSON)
			book.Authors = &s
		}
	}

	if pubRaw, ok := olResp["publishers"]; ok {
		switch p := pubRaw.(type) {
		case string:
			if p != "" {
				book.Publisher = &p
			}
		case []interface{}:
			if len(p) > 0 {
				if pub, ok := p[0].(string); ok && pub != "" {
					book.Publisher = &pub
				}
			}
		}
	}

	if publishDate, ok := olResp["publish_date"].(string); ok {
		if m := yearRe.FindStringSubmatch(publishDate); m != nil {
			year, _ := strconv.Atoi(m[1])
			book.PublicationYear = &year
		}
	}

	if pagesRaw, ok := olResp["number_of_pages"].(float64); ok && pagesRaw > 0 {
		pages := int(pagesRaw)
		book.PageCount = &pages
	}

	if subjectsRaw, ok := olResp["subjects"].([]interface{}); ok && len(subjectsRaw) > 0 {
		var subjects []string
		for _, s := range subjectsRaw {
			if subject, ok := s.(string); ok && subject != "" {
				subjects = append(subjects, subject)
			}
		}
		if len(subjects) > 0 {
			subjectsJSON, _ := json.Marshal(subjects)
			s := string(subjectsJSON)
			book.Genres = &s
		}
	}

	extractStringSlice := func(field string) *string {
		if raw, ok := olResp[field].([]interface{}); ok && len(raw) > 0 {
			var vals []string
			for _, v := range raw {
				if s, ok := v.(string); ok && s != "" {
					vals = append(vals, s)
				}
			}
			if len(vals) > 0 {
				b, _ := json.Marshal(vals)
				s := string(b)
				return &s
			}
		}
		return nil
	}
	book.SubjectPlaces = extractStringSlice("subject_places")
	book.SubjectPeople = extractStringSlice("subject_people")
	book.SubjectTimes = extractStringSlice("subject_times")

	if illustratorsRaw, ok := olResp["illustrators"].([]interface{}); ok && len(illustratorsRaw) > 0 {
		illustrators := resolveOpenLibraryAuthorKeys(illustratorsRaw)
		if len(illustrators) > 0 {
			illustratorsJSON, _ := json.Marshal(illustrators)
			s := string(illustratorsJSON)
			book.Illustrators = &s
		}
	}

	var coverID float64
	if ci, ok := olResp["cover_i"].(float64); ok && ci > 0 {
		coverID = ci
	} else if coversRaw, ok := olResp["covers"].([]interface{}); ok && len(coversRaw) > 0 {
		if firstCover, ok := coversRaw[0].(float64); ok && firstCover > 0 {
			coverID = firstCover
		}
	}
	if coverID > 0 {
		coverURL := fmt.Sprintf("%s/b/id/%d-L.jpg", olConfig.CoversURL, int(coverID))
		book.CoverImageURL = &coverURL
	}

	if book.CoverImageURL == nil {
		if workRaw, ok := olResp["work"].(map[string]interface{}); ok {
			if workKey, ok := workRaw["key"].(string); ok && workKey != "" {
				workKeyParts := strings.Split(workKey, "/")
				if len(workKeyParts) > 0 {
					olid := workKeyParts[len(workKeyParts)-1]
					coverURL := fmt.Sprintf("%s/b/olid/%s-L.jpg", olConfig.CoversURL, olid)
					book.CoverImageURL = &coverURL
				}
			}
		}
	}

	if ddcRaw, ok := olResp["dewey_decimal_class"]; ok {
		switch d := ddcRaw.(type) {
		case string:
			if d != "" {
				book.DeweyDecimalClass = &d
			}
		case []interface{}:
			if len(d) > 0 {
				if s, ok := d[0].(string); ok && s != "" {
					book.DeweyDecimalClass = &s
				}
			}
		}
	}

	if descRaw, ok := olResp["description"]; ok {
		switch d := descRaw.(type) {
		case string:
			if d != "" {
				book.Description = &d
			}
		case map[string]interface{}:
			if v, ok := d["value"].(string); ok && v != "" {
				book.Description = &v
			}
		}
	}

	if langRaw, ok := olResp["language"]; ok {
		switch l := langRaw.(type) {
		case string:
			if l != "" {
				book.Language = &l
			}
		case []interface{}:
			if len(l) > 0 {
				if s, ok := l[0].(string); ok && s != "" {
					book.Language = &s
				}
			}
		}
	}

	return book, "open_library", nil
}

// resolveOpenLibraryAuthorKeys resolves author entries from the Open Library API.
func resolveOpenLibraryAuthorKeys(raw []interface{}) []string {
	var named []string
	var keysToResolve []string

	for _, a := range raw {
		if m, ok := a.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok && name != "" {
				named = append(named, name)
			} else if key, ok := m["key"].(string); ok && key != "" {
				keysToResolve = append(keysToResolve, key)
			}
		}
	}

	if len(keysToResolve) > 0 {
		results := make(chan string, len(keysToResolve))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 3)
		for _, key := range keysToResolve {
			sem <- struct{}{}
			wg.Add(1)
			go func(k string) {
				defer func() {
					<-sem
					wg.Done()
				}()
				if name := fetchOpenLibraryAuthorName(k); name != "" {
					results <- name
				} else {
					slog.Warn("failed to resolve Open Library author key", "key", k)
				}
			}(key)
		}
		wg.Wait()
		close(results)
		for name := range results {
			named = append(named, name)
		}
	}

	return named
}

// fetchOpenLibraryAuthorName fetches the name of a single author by their OL key.
func fetchOpenLibraryAuthorName(key string) string {
	u := fmt.Sprintf("%s%s.json", olConfig.BaseURL, key)
	body, statusCode, err := olRequestWithRetry(context.Background(), u, 2)
	if err != nil {
		slog.Warn("Open Library author request failed after retries", "key", key, "error", err)
		return ""
	}
	if statusCode != http.StatusOK {
		slog.Warn("Open Library author request failed", "key", key, "status", statusCode)
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		slog.Warn("failed to parse Open Library author response", "key", key, "error", err)
		return ""
	}

	if name, ok := data["name"].(string); ok {
		return name
	}
	slog.Warn("Open Library author response missing name field", "key", key)
	return ""
}

// toString safely converts an interface{} to string.
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
