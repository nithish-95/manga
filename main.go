package main

import (
	"io"
	"log"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nithish-95/manga/mangadex"
)

func main() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	r.Get("/", homeHandler)
	r.Get("/image-proxy", imageProxyHandler)
	r.Get("/manga/{mangaID}", mangaHandler)
	r.Get("/read/{chapterID}", chapterHandler)

	// Serve static files (if any)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("public/static"))))

	http.ListenAndServe(":3001", r)
}

// homeHandler searches for manga based on a query parameter.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("search")

	mangas, err := mangadex.SearchManga(searchQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// For each manga, try to get its cover image.
	for i, m := range mangas {
		coverURL, err := mangadex.GetCoverForManga(m.ID)
		if err != nil {
			log.Printf("Error fetching cover for manga %s: %v", m.ID, err)
		} else {
			mangas[i].Attributes.CoverURL = coverURL
		}
	}

	tmpl := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/home.html",
	))
	tmpl.Execute(w, map[string]interface{}{
		"Mangas": mangas,
	})
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

	// Fetch the manga details.
	manga, err := mangadex.GetManga(mangaID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the chapters for this manga.
	chapters, err := mangadex.GetChaptersForManga(mangaID)
	if err != nil {
		log.Printf("Error fetching chapters for manga %s: %v", mangaID, err)
		// If error, continue with an empty slice.
		chapters = []mangadex.ChapterData{}
	}

	// Prepare the data to pass to the template.
	data := struct {
		Manga    mangadex.Manga
		Chapters []mangadex.ChapterData
	}{
		Manga:    manga,
		Chapters: chapters,
	}

	tmpl := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/manga.html",
	))
	tmpl.Execute(w, data)
}

// chapterHandler fetches and displays a chapter for reading.
func chapterHandler(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")
	chapter, err := mangadex.GetChapter(chapterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/reader.html",
	))
	tmpl.Execute(w, chapter)
}
