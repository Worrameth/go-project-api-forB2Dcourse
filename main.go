package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Book struct {
	BookID    int    `json:"bookid"`
	BookName  string `json:"bookname"`
	Author    string `json:"author"`
	Genre     string `json:"genre"`
	Publisher string `json:"publisher"`
}

const bookPath = "books"

var Db *sql.DB

const basePath = "/api"

func getBookList() ([]Book, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := Db.QueryContext(ctx, `SELECT * FROM books`)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer results.Close()
	books := make([]Book, 0)
	for results.Next() {
		var book Book
		results.Scan(
			&book.BookID,
			&book.BookName,
			&book.Author,
			&book.Genre,
			&book.Publisher)
		books = append(books, book)
	}
	return books, nil
}

func getBook(bookID int) (*Book, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := Db.QueryRowContext(ctx, `SELECT * FROM books WHERE bookid = ?`, bookID)

	book := &Book{}
	err := row.Scan(
		&book.BookID,
		&book.BookName,
		&book.Author,
		&book.Genre,
		&book.Publisher,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		log.Println(err)
		return nil, err
	}

	return book, nil
}

func removeBook(bookID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := Db.ExecContext(ctx, `DELETE FROM books WHERE bookid = ?`, bookID)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func insertBook(book Book) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := Db.ExecContext(ctx, `INSERT INTO books (bookid, bookname, author, genre, publisher) VALUES (?,?,?,?,?)`, book.BookID, book.BookName, book.Author, book.Genre, book.Publisher)
	if err != nil {
		log.Println(err.Error())
		return 0, err
	}
	insertID, err := result.LastInsertId()
	if err != nil {
		log.Println(err.Error())
		return 0, err
	}
	return int(insertID), nil
}

func handleBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		bookList, err := getBookList()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		j, err := json.Marshal(bookList)
		if err != nil {
			log.Fatal(err)
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
	case http.MethodPost:
		var book Book
		err := json.NewDecoder(r.Body).Decode(&book)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = insertBook(book)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		//w.Write([]byte(fmt.Sprintf(`{"bookid":%d}`, BookID)))
	case http.MethodOptions:
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleBook(w http.ResponseWriter, r *http.Request) {
	urlPathSegments := strings.Split(r.URL.Path, fmt.Sprintf("%s/", bookPath))
	if len(urlPathSegments[1:]) > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bookID, err := strconv.Atoi(urlPathSegments[len(urlPathSegments)-1])
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		book, err := getBook(bookID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if book == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		j, err := json.Marshal(book)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
	case http.MethodDelete:
		err := removeBook(bookID)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Origin, X-Requested-With")
		handler.ServeHTTP(w, r)

	})
}

func SetupRoutes(apiBasePath string) {

	BooksHandler := http.HandlerFunc(handleBooks)
	http.Handle(fmt.Sprintf("%s/%s", apiBasePath, bookPath), corsMiddleware(BooksHandler))

	bookHandler := http.HandlerFunc(handleBook)
	http.Handle(fmt.Sprintf("%s/%s/", apiBasePath, bookPath), corsMiddleware(bookHandler))

}

func SetupDB() {
	var err error
	Db, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/bookdb")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(Db)
	Db.SetConnMaxLifetime(time.Minute * 3)
	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)
}

func main() {
	SetupDB()
	SetupRoutes(basePath)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
