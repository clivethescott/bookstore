package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/clivethescott/bookstore/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var env *models.DBEnv

// BookRepository implements persistence for books
type BookRepository interface {
	GetBooks(ctx context.Context) ([]*models.Book, error)
	GetBookByIsbn(ctx context.Context, isbn string) (*models.Book, error)
	CreateBook(ctx context.Context, book *models.Book) error
}

func init() {
	db := sqlx.MustOpen("mysql", "root:@unix(/tmp/mysql.sock)/bookstore")
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		log.Printf("failed to connect to the db: %v\n", err)
	}

	env = &models.DBEnv{DB: db}
}

func main() {
	var repo BookRepository = env
	r := chi.NewRouter()
	r.Use(acceptJSON, middleware.RedirectSlashes)
	r.MethodFunc(http.MethodGet, "/book", books(repo))
	r.MethodFunc(http.MethodPost, "/book", createBook(repo))
	r.MethodFunc(http.MethodGet, "/book/{isbn}", bookByIsbn(repo))

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
	log.Printf("internal server error: %v\n", err)
	errCode := http.StatusInternalServerError
	http.Error(w, http.StatusText(errCode), errCode)
}

func createBook(repo BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		req := new(models.Book)
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(req); err != nil {
			badRequest(err, w)
			return
		}

		if err := repo.CreateBook(r.Context(), req); err != nil {

			switch {
			case errors.Is(err, models.ErrBookExists):
				http.Error(w, "book already exists", http.StatusBadRequest)
			case errors.Is(err, models.ErrInvalidBook):
				http.Error(w, "book missing info", http.StatusBadRequest)
			default:
				serverError(err, w)
			}

			return
		}

		book, err := repo.GetBookByIsbn(r.Context(), req.Isbn)
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
}

func bookByIsbn(repo BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		isbn := chi.URLParam(r, "isbn")
		book, err := repo.GetBookByIsbn(r.Context(), isbn)

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
}

func books(repo BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		books, err := repo.GetBooks(r.Context())
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
}
