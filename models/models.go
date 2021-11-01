package models

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// DBEnv holds DB environment deps
type DBEnv struct {
	DB *sqlx.DB
}

// Book represents a book
type Book struct {
	Isbn   string  `db:"isbn"`
	Title  string  `db:"title"`
	Author string  `db:"author"`
	Price  float32 `db:"price"`
}

var (
	// ErrBookNotFound is returned when a book isn't found
	ErrBookNotFound = errors.New("book not found")

	// ErrBookExists is returned when a matching book exists
	ErrBookExists = errors.New("book already exists")
)

func (b Book) String() string {
	return fmt.Sprintf("Book(isbn=%s, title=%s, author=%s, price=$%.2f)",
		b.Isbn, b.Title, b.Author, b.Price)
}

// GetBooks find all books
func (env *DBEnv) GetBooks() ([]*Book, error) {

	books := []*Book{}
	err := env.DB.Select(&books, "SELECT isbn, title, author, price from books")
	if err != nil {
		return nil, err
	}
	return books, nil
}

// GetBookByIsbn finds a single book by its ISBN
func (env *DBEnv) GetBookByIsbn(isbn string) (*Book, error) {
	book := new(Book)
	err := env.DB.Get(book, "SELECT isbn, title, author, price from books WHERE isbn = $1", isbn)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBookNotFound
		}
		return nil, err
	}
	return book, nil
}

func (env *DBEnv) hasBookByIsbn(isbn string) (bool, error) {

	_, err := env.GetBookByIsbn(isbn)
	if err == nil {
		return true, nil
	}

	return false, err
}

// CreateBook creates a new book
func (env *DBEnv) CreateBook(req *Book) error {
	if req.Isbn == "" || req.Title == "" || req.Author == "" {
		return errors.New("all fields are required")
	}

	exists, err := env.hasBookByIsbn(req.Isbn)
	if err != nil {
		return err
	}

	if exists {
		return ErrBookExists
	}

	result, err := env.DB.NamedExec(`INSERT INTO books(isbn, title, author, price) 
	VALUES(:isbn, :title, :author, :price)`, req)
	if err != nil {
		return err
	}

	if _, err = result.RowsAffected(); err != nil {
		return err
	}

	return nil
}
