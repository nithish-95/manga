package mangadex

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	apiBase      = "https://api.mangadex.org"
	coverBaseURL = "https://uploads.mangadex.org/covers"
)

var (
	mangaCache   = NewCache[Manga]()
	chapterCache = NewCache[*ChaptersResponse]()
)

// Manga represents a manga from the API.
type Manga struct {
	ID         string `json:"id"`
	Attributes struct {
		Title       map[string]string `json:"title"`
		Description map[string]string `json:"description"`
		// CoverURL will be populated after fetching cover data.
		CoverURL string `json:"cover_url,omitempty"`
	} `json:"attributes"`
	Chapters []ChapterSummary `json:"chapters"` // Consider removing if unused.
}

// ChapterSummary represents basic chapter info.
type ChapterSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Chapter represents a full chapter with pages.
type Chapter struct {
	ID         string `json:"id"`
	Attributes struct {
		Title string `json:"title"`
	} `json:"attributes"`
}

// CoverData represents one cover entry returned by the API.
type CoverData struct {
	ID         string `json:"id"`
	Attributes struct {
		FileName string `json:"fileName"`
	} `json:"attributes"`
	Relationships []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"relationships"`
}

// CoverResponse is the API response for cover requests.
type CoverResponse struct {
	Result string      `json:"result"`
	Data   []CoverData `json:"data"`
}

// ChaptersResponse represents the response from the chapter endpoint.
type ChaptersResponse struct {
	Result string        `json:"result"`
	Data   []ChapterData `json:"data"`
	Total  int           `json:"total"`
}

// ChapterData represents a chapter entry from the chapter endpoint.
type ChapterData struct {
	ID         string `json:"id"`
	Attributes struct {
		Chapter string `json:"chapter"` // chapter number as string (may be empty)
		Title   string `json:"title"`
		Volume  string `json:"volume"`
	} `json:"attributes"`
}

// AtHomeServerResponse represents the response from the /at-home/server/{chapter_id} endpoint.
type AtHomeServerResponse struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash      string   `json:"hash"`
		Data      []string `json:"data"`
		DataSaver []string `json:"dataSaver"`
	} `json:"chapter"`
}

// httpClient with timeout for requests.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

type MangaListResponse struct {
	Result string  `json:"result"`
	Data   []Manga `json:"data"`
	Total  int     `json:"total"`
}

// GetMangaList fetches a list of manga based on provided parameters.
func GetMangaList(params url.Values) ([]Manga, error) {
	requestURL := fmt.Sprintf("%s/manga?%s", apiBase, params.Encode())
	log.Printf("Requesting URL: %s", requestURL)
	resp, err := httpClient.Get(requestURL)
	if err != nil {
		log.Printf("HTTP GET error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("Response Status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-OK status: %s", resp.Status)
	}

	var result MangaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("JSON decode error: %v", err)
		return nil, err
	}
	return result.Data, nil
}

// SearchManga searches for manga by title.
func SearchManga(title string) ([]Manga, error) {
	params := url.Values{}
	params.Add("title", title)
	return GetMangaList(params)
}

// GetPopularManga fetches popular manga.
func GetPopularManga() ([]Manga, error) {
	params := url.Values{}
	params.Add("order[followedCount]", "desc")
	params.Add("limit", "10") // Fetch top 10 popular manga
	return GetMangaList(params)
}

// GetRecentlyUpdatedManga fetches recently updated manga.
func GetRecentlyUpdatedManga() ([]Manga, error) {
	params := url.Values{}
	params.Add("order[updatedAt]", "desc")
	params.Add("limit", "10") // Fetch 10 recently updated manga
	return GetMangaList(params)
}

// GetRandomManga fetches a random manga.
func GetRandomManga() (Manga, error) {
	requestURL := fmt.Sprintf("%s/manga/random", apiBase)
	log.Printf("Requesting URL: %s", requestURL)
	resp, err := httpClient.Get(requestURL)
	if err != nil {
		log.Printf("HTTP GET error: %v", err)
		return Manga{}, err
	}
	defer resp.Body.Close()

	log.Printf("Response Status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return Manga{}, fmt.Errorf("API returned non-OK status: %s", resp.Status)
	}

	var result struct {
		Data Manga `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("JSON decode error: %v", err)
		return Manga{}, err
	}
	return result.Data, nil
}

// GetManga fetches a specific manga by its ID.
func GetManga(mangaID string) (Manga, error) {
	if manga, ok := mangaCache.Get(mangaID); ok {
		return manga, nil
	}

	url := fmt.Sprintf("%s/manga/%s", apiBase, mangaID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return Manga{}, err
	}
	defer resp.Body.Close()

	var result struct {
		Data Manga `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Manga{}, err
	}

	mangaCache.Set(mangaID, result.Data)
	return result.Data, nil
}

// GetChapterDetails fetches a specific chapter by its ID.
func GetChapterDetails(chapterID string) (Chapter, error) {
	url := fmt.Sprintf("%s/chapter/%s", apiBase, chapterID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return Chapter{}, err
	}
	defer resp.Body.Close()

	var result struct {
		Data Chapter `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Chapter{}, err
	}
	return result.Data, nil
}

// GetChapterPages fetches the pages for a specific chapter by its ID.
func GetChapterPages(chapterID string) ([]string, error) {
	url := fmt.Sprintf("%s/at-home/server/%s", apiBase, chapterID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result AtHomeServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var pages []string
	for _, page := range result.Chapter.Data {
		pages = append(pages, fmt.Sprintf("%s/data/%s/%s", result.BaseURL, result.Chapter.Hash, page))
	}

	return pages, nil
}

// GetCoverForManga fetches the cover data for a specific manga ID and returns the cover URL.
// It calls the cover endpoint using a filter for the manga ID.
func GetCoverForManga(mangaID string) (string, error) {
	// The API supports filtering by manga id: ?manga[]=<mangaID>
	url := fmt.Sprintf("%s/cover?manga[]=%s", apiBase, mangaID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var coverResp CoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&coverResp); err != nil {
		return "", err
	}

	// If at least one cover is returned, construct the URL.
	if len(coverResp.Data) > 0 {
		cover := coverResp.Data[0]
		coverURL := fmt.Sprintf("%s/%s/%s", coverBaseURL, mangaID, cover.Attributes.FileName)
		return coverURL, nil
	}

	return "", nil // No cover found.
}

// GetMangaChapters fetches all chapters for a specific manga ID.
func GetMangaChapters(mangaID string, limit, offset int) (*ChaptersResponse, error) {
	cacheKey := fmt.Sprintf("%s-%d-%d", mangaID, limit, offset)
	if chapters, ok := chapterCache.Get(cacheKey); ok {
		return chapters, nil
	}

	baseURL, _ := url.Parse(fmt.Sprintf("%s/manga/%s/feed", apiBase, mangaID))
	params := url.Values{}
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("offset", fmt.Sprintf("%d", offset))
	params.Add("translatedLanguage[]", "en")
	params.Add("order[chapter]", "asc")
	baseURL.RawQuery = params.Encode()

	resp, err := httpClient.Get(baseURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chaptersResp ChaptersResponse
	if err := json.NewDecoder(resp.Body).Decode(&chaptersResp); err != nil {
		return nil, err
	}

	chapterCache.Set(cacheKey, &chaptersResp)
	return &chaptersResp, nil
}

// GetChaptersForManga fetches chapters for a specific manga ID.
func GetChaptersForManga(mangaID string, limit, offset int) (*ChaptersResponse, error) {
	return GetMangaChapters(mangaID, limit, offset)
}