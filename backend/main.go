package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	// Pre-parse all templates on application startup.
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	templates = make(map[string]*template.Template)
	templates["home"] = template.Must(template.New("home.html").Funcs(funcMap).ParseFiles("../frontend/templates/base.html", "../frontend/templates/home.html"))
	templates["manga"] = template.Must(template.New("manga.html").Funcs(funcMap).ParseFiles("../frontend/templates/base.html", "../frontend/templates/manga.html"))
	templates["reader"] = template.Must(template.New("reader.html").Funcs(funcMap).ParseFiles("../frontend/templates/base.html", "../frontend/templates/reader.html"))
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

	// Serve static files (if any)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("../frontend/public"))))

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
