package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/clivethescott/bookstore/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var env *models.DBEnv

func init() {
	db := sqlx.MustOpen("postgres", "user=clive dbname=bookstore sslmode=disable")
	db.SetMaxOpenConns(5)
	if err := db.Ping(); err != nil {
		log.Printf("failed to connect to the db: %v\n", err)
	}

	env = &models.DBEnv{DB: db}
}

func main() {
	r := chi.NewRouter()
	r.Use(acceptJSON, middleware.RedirectSlashes)
	r.MethodFunc(http.MethodGet, "/book", books)
	r.MethodFunc(http.MethodPost, "/book", createBook)
	r.MethodFunc(http.MethodGet, "/book/{isbn}", bookByIsbn)

	server := &http.Server{
		Addr:         ":3000",
		Handler:      r,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	server.ListenAndServe()
}

func acceptJSON(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("accept"); !strings.Contains(accept, "json") {
			http.Error(w, "only json is supported", http.StatusUnsupportedMediaType)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func badRequest(err error, w http.ResponseWriter) {
	log.Printf("bad request: %\n", err)
	errCode := http.StatusBadRequest
	http.Error(w, http.StatusText(errCode), errCode)
}

func serverError(err error, w http.ResponseWriter) {
	log.Printf("internal server error: %\n", err)
	errCode := http.StatusInternalServerError
	http.Error(w, http.StatusText(errCode), errCode)
}

func createBook(w http.ResponseWriter, r *http.Request) {
	req := new(models.Book)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(req); err != nil {
		badRequest(err, w)
		return
	}

	if err := env.CreateBook(r.Context(), req); err != nil {
		if err == models.ErrBookExists {
			http.Error(w, "book already exists", http.StatusBadRequest)
			return
		}
		serverError(err, w)
		return
	}

	book, err := env.GetBookByIsbn(r.Context(), req.Isbn)
	if err != nil {
		serverError(err, w)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(book); err != nil {
		serverError(err, w)
	}

}

func bookByIsbn(w http.ResponseWriter, r *http.Request) {
	isbn := chi.URLParam(r, "isbn")
	book, err := env.GetBookByIsbn(r.Context(), isbn)

	if err != nil {
		if err == models.ErrBookNotFound {
			http.Error(w, "book not found by isbn "+isbn, http.StatusNotFound)
		} else {
			serverError(err, w)
		}
		return
	}

	w.Header().Set("content-type", "application/json")
	encoder := json.NewEncoder(w)
	if err = encoder.Encode(book); err != nil {
		serverError(err, w)
		return
	}

}

func books(w http.ResponseWriter, r *http.Request) {

	books, err := env.GetBooks(r.Context())
	if err != nil {
		serverError(err, w)
		return
	}

	w.Header().Set("content-type", "application/json")
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(books); err != nil {
		serverError(err, w)
	}
}
