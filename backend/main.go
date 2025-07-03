package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nithish-95/manga/backend/mangadex"
)

var (
	// templates holds all our parsed templates.
	templates map[string]*template.Template
)

func init() {
	// Get the absolute path of the executable.
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	// The executable is in bin/, so the project root is one level up.
	projectRoot := filepath.Dir(filepath.Dir(executablePath))

	// Pre-parse all templates on application startup.
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	templates = make(map[string]*template.Template)
	templates["home"] = template.Must(template.New("home.html").Funcs(funcMap).ParseFiles(filepath.Join(projectRoot, "frontend/templates/base.html"), filepath.Join(projectRoot, "frontend/templates/home.html")))
	templates["manga"] = template.Must(template.New("manga.html").Funcs(funcMap).ParseFiles(filepath.Join(projectRoot, "frontend/templates/base.html"), filepath.Join(projectRoot, "frontend/templates/manga.html")))
	templates["reader"] = template.Must(template.New("reader.html").Funcs(funcMap).ParseFiles(filepath.Join(projectRoot, "frontend/templates/base.html"), filepath.Join(projectRoot, "frontend/templates/reader.html")))
	templates["manga_list"] = template.Must(template.New("manga_list.html").Funcs(funcMap).ParseFiles(filepath.Join(projectRoot, "frontend/templates/base.html"), filepath.Join(projectRoot, "frontend/templates/manga_list.html")))
}

func main() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	r.Get("/", homeHandler)
	r.Get("/image-proxy", imageProxyHandler)
	r.Get("/manga/{mangaID}", mangaHandler)
	r.Get("/manga/{mangaID}/read/{chapterID}", chapterHandler)
	r.Get("/popular", popularMangaHandler)
	r.Get("/recent", recentMangaHandler)
	r.Get("/random-manga-json", randomMangaJSONHandler)

	// Get the absolute path of the executable.
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	// The executable is in bin/, so the project root is one level up.
	projectRoot := filepath.Dir(filepath.Dir(executablePath))
	staticPath := filepath.Join(projectRoot, "frontend", "public")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Println("Starting server on :" + port)
	http.ListenAndServe(":"+port, r)
}

// homeHandler searches for manga based on a query parameter.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("search")

	data := struct {
		Mangas           []mangadex.Manga
		SearchQuery      string
		PopularMangas    []mangadex.Manga
		RecentMangas     []mangadex.Manga
		RandomManga      mangadex.Manga
		PrevPage         int
		NextPage         int
		TotalPages       int
	}{
		SearchQuery: searchQuery,
		PrevPage:    0,
		NextPage:    0,
		TotalPages:  0,
	}

	var err error

	if searchQuery != "" {
		data.Mangas, err = mangadex.SearchManga(searchQuery)
		if err != nil {
			log.Printf("Error searching manga: %v", err)
			http.Error(w, "Error searching manga", http.StatusInternalServerError)
			return
		}

		// Fetch covers for search results
		var coverWg sync.WaitGroup
		for i := range data.Mangas {
			coverWg.Add(1)
			go func(i int) {
				defer coverWg.Done()
				coverURL, coverErr := mangadex.GetCoverForManga(data.Mangas[i].ID)
				if coverErr != nil {
					log.Printf("Error fetching cover for search result manga %s: %v", data.Mangas[i].ID, coverErr)
				} else {
					data.Mangas[i].Attributes.CoverURL = coverURL
				}
			}(i)
		}
		coverWg.Wait()

	} else {
		var wg sync.WaitGroup
		var popularErr, recentErr, randomErr error

		wg.Add(1)
		go func() {
			defer wg.Done()
			data.PopularMangas, popularErr = mangadex.GetPopularManga()
			if popularErr != nil {
				log.Printf("Error fetching popular mangas: %v", popularErr)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			data.RecentMangas, recentErr = mangadex.GetRecentlyUpdatedManga()
			if recentErr != nil {
				log.Printf("Error fetching recently updated mangas: %v", recentErr)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			data.RandomManga, randomErr = mangadex.GetRandomManga()
			if randomErr != nil {
				log.Printf("Error fetching random manga: %v", randomErr)
			}
		}()

		wg.Wait()

		// Fetch covers for popular mangas
		var coverWg sync.WaitGroup
		for i := range data.PopularMangas {
			coverWg.Add(1)
			go func(i int) {
				defer coverWg.Done()
				coverURL, coverErr := mangadex.GetCoverForManga(data.PopularMangas[i].ID)
				if coverErr != nil {
					log.Printf("Error fetching cover for popular manga %s: %v", data.PopularMangas[i].ID, coverErr)
				} else {
					data.PopularMangas[i].Attributes.CoverURL = coverURL
				}
			}(i)
		}
		coverWg.Wait()

		// Fetch covers for recent mangas
		for i := range data.RecentMangas {
			coverWg.Add(1)
			go func(i int) {
				defer coverWg.Done()
				coverURL, coverErr := mangadex.GetCoverForManga(data.RecentMangas[i].ID)
				if coverErr != nil {
					log.Printf("Error fetching cover for recent manga %s: %v", data.RecentMangas[i].ID, coverErr)
				} else {
					data.RecentMangas[i].Attributes.CoverURL = coverURL
				}
			}(i)
		}
		coverWg.Wait()

		// Fetch cover for random manga
		if data.RandomManga.ID != "" {
			coverURL, coverErr := mangadex.GetCoverForManga(data.RandomManga.ID)
			if coverErr != nil {
				log.Printf("Error fetching cover for random manga %s: %v", data.RandomManga.ID, coverErr)
			} else {
				data.RandomManga.Attributes.CoverURL = coverURL
			}
		}
	}

	err = templates["home"].ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// imageProxyHandler proxies image requests.
func imageProxyHandler(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	resp, err := http.Get(imageURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

// mangaHandler fetches and displays a single manga's details along with its chapters.
func mangaHandler(w http.ResponseWriter, r *http.Request) {
	mangaID := chi.URLParam(r, "mangaID")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit := 10
	offset := (page - 1) * limit

	// Fetch the manga details.
	manga, err := mangadex.GetManga(mangaID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch cover for the manga
	if manga.ID != "" {
		coverURL, coverErr := mangadex.GetCoverForManga(manga.ID)
		if coverErr != nil {
			log.Printf("Error fetching cover for manga %s: %v", manga.ID, coverErr)
		} else {
			manga.Attributes.CoverURL = coverURL
		}
	}

	// Fetch the chapters for this manga.
	chaptersResp, err := mangadex.GetChaptersForManga(mangaID, limit, offset)
	if err != nil {
		log.Printf("Error fetching chapters for manga %s: %v", mangaID, err)
		// If error, continue with an empty slice.
		chaptersResp = &mangadex.ChaptersResponse{}
	}

	// Prepare the data to pass to the template.
	data := struct {
		Manga      mangadex.Manga
		Chapters   []mangadex.ChapterData
		Total      int
		Page       int
		Limit      int
		TotalPages int
		BackLink   string
	}{
		Manga:      manga,
		Chapters:   chaptersResp.Data,
		Total:      chaptersResp.Total,
		Page:       page,
		Limit:      limit,
		TotalPages: (chaptersResp.Total + limit - 1) / limit,
		BackLink:   "/",
	}

	err = templates["manga"].ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// chapterHandler fetches and displays a chapter for reading.
func chapterHandler(w http.ResponseWriter, r *http.Request) {
	mangaID := chi.URLParam(r, "mangaID")
	chapterID := chi.URLParam(r, "chapterID")

	chapter, err := mangadex.GetChapterDetails(chapterID)
	if err != nil {
		log.Printf("Error getting chapter details for %s: %v", chapterID, err)
		http.Error(w, "Failed to get chapter details", http.StatusInternalServerError)
		return
	}

	pages, err := mangadex.GetChapterPages(chapterID)
	if err != nil {
		log.Printf("Error getting chapter pages for %s: %v", chapterID, err)
		http.Error(w, "Failed to get chapter pages", http.StatusInternalServerError)
		return
	}

	chapters, err := mangadex.GetMangaChapters(mangaID, 100, 0)
	if err != nil {
		log.Printf("Error fetching chapters for manga %s: %v", mangaID, err)
	}

	var prevChapter, nextChapter string
	if chapters != nil {
		for i, c := range chapters.Data {
			if c.ID == chapterID {
				if i > 0 {
					prevChapter = chapters.Data[i-1].ID
				}
				if i < len(chapters.Data)-1 {
					nextChapter = chapters.Data[i+1].ID
				}
				break
			}
		}
	}

	data := struct {
		Chapter     mangadex.Chapter
		Pages       []string
		MangaID     string
		PrevChapter string
		NextChapter string
		BackLink    string
	}{
		Chapter:     chapter,
		Pages:       pages,
		MangaID:     mangaID,
		PrevChapter: prevChapter,
		NextChapter: nextChapter,
		BackLink:    fmt.Sprintf("/manga/%s", mangaID),
	}

	err = templates["reader"].ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func popularMangaHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit := 20 // Display more on the dedicated page
	offset := (page - 1) * limit

	mangas, err := mangadex.GetPopularMangaWithPagination(limit, offset)
	if err != nil {
		log.Printf("Error fetching popular mangas: %v", err)
		http.Error(w, "Error fetching popular mangas", http.StatusInternalServerError)
		return
	}

	// Fetch covers for mangas
	var coverWg sync.WaitGroup
	for i := range mangas {
		coverWg.Add(1)
		go func(i int) {
			defer coverWg.Done()
			coverURL, coverErr := mangadex.GetCoverForManga(mangas[i].ID)
			if coverErr != nil {
				log.Printf("Error fetching cover for manga %s: %v", mangas[i].ID, coverErr)
			} else {
				mangas[i].Attributes.CoverURL = coverURL
			}
		}(i)
	}
	coverWg.Wait()

	data := struct {
		Title      string
		Mangas     []mangadex.Manga
		BaseURL    string
		PrevPage   int
		NextPage   int
		TotalPages int
	}{
		Title:    "Popular Mangas",
		Mangas:   mangas,
		BaseURL:  "/popular",
		PrevPage: page - 1,
		NextPage: page + 1,
		// TotalPages: calculate based on total count from API if available
	}

	err = templates["manga_list"].ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func recentMangaHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit := 20 // Display more on the dedicated page
	offset := (page - 1) * limit

	mangas, err := mangadex.GetRecentlyUpdatedMangaWithPagination(limit, offset)
	if err != nil {
		log.Printf("Error fetching recently updated mangas: %v", err)
		http.Error(w, "Error fetching recently updated mangas", http.StatusInternalServerError)
		return
	}

	// Fetch covers for mangas
	var coverWg sync.WaitGroup
	for i := range mangas {
		coverWg.Add(1)
		go func(i int) {
			defer coverWg.Done()
			coverURL, coverErr := mangadex.GetCoverForManga(mangas[i].ID)
			if coverErr != nil {
				log.Printf("Error fetching cover for manga %s: %v", mangas[i].ID, coverErr)
			} else {
				mangas[i].Attributes.CoverURL = coverURL
			}
		}(i)
	}
	coverWg.Wait()

	data := struct {
		Title      string
		Mangas     []mangadex.Manga
		BaseURL    string
		PrevPage   int
		NextPage   int
		TotalPages int
	}{
		Title:    "Recently Updated Mangas",
		Mangas:   mangas,
		BaseURL:  "/recent",
		PrevPage: page - 1,
		NextPage: page + 1,
		// TotalPages: calculate based on total count from API if available
	}

	err = templates["manga_list"].ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func randomMangaJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	manga, err := mangadex.GetRandomManga()
	if err != nil {
		log.Printf("Error fetching random manga: %v", err)
		http.Error(w, "Failed to fetch random manga", http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch random manga"})
		return
	}

	// Fetch cover for random manga
	if manga.ID != "" {
		coverURL, coverErr := mangadex.GetCoverForManga(manga.ID)
		if coverErr != nil {
			log.Printf("Error fetching cover for random manga %s: %v", manga.ID, coverErr)
			// Continue without cover if there's an error fetching it
		} else {
			manga.Attributes.CoverURL = coverURL
		}
	}

	// If manga is empty (e.g., no random manga found), return an empty object
	if manga.ID == "" {
		json.NewEncoder(w).Encode(map[string]string{})
		return
	}

	json.NewEncoder(w).Encode(manga)
}