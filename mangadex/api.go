package mangadex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	apiBase      = "https://api.mangadex.org"
	coverBaseURL = "https://uploads.mangadex.org/covers"
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
		Title string   `json:"title"`
		Pages []string `json:"pages"`
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
	// Additional pagination fields can be added here if needed.
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

// httpClient with timeout for requests.
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// SearchManga searches for manga by title.
func SearchManga(title string) ([]Manga, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/manga?title=%s", apiBase, title))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []Manga `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetManga fetches a specific manga by its ID.
func GetManga(mangaID string) (Manga, error) {
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
	return result.Data, nil
}

// GetChapter fetches a specific chapter by its ID.
func GetChapter(chapterID string) (Chapter, error) {
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

// GetChaptersForManga fetches chapters for a specific manga ID.
func GetChaptersForManga(mangaID string) ([]ChapterData, error) {
	// Build the URL using the provided curl parameters and adding filter[manga]=<mangaID>
	url := fmt.Sprintf("%s/chapter?filter[manga]=%s&limit=10&contentRating[]=safe&contentRating[]=suggestive&contentRating[]=erotica&includeFutureUpdates=1&order[createdAt]=asc&order[updatedAt]=asc&order[publishAt]=asc&order[readableAt]=asc&order[volume]=asc&order[chapter]=asc", apiBase, mangaID)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chaptersResp ChaptersResponse
	if err := json.NewDecoder(resp.Body).Decode(&chaptersResp); err != nil {
		return nil, err
	}
	return chaptersResp.Data, nil
}
