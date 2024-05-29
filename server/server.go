package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Resp struct {
	USDBRL struct {
		Bid string `json:"bid"`
	} `json:"USDBRL"`
}

type Cotacao struct {
	Cotacao string `json:"cotacaoDolar"`
}

func main() {
	db, err := sql.Open("sqlite3", "./cotacao.db")
	if err != nil {
		log.Println("err")
	}
	defer db.Close()

	createTable(db)

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		handlerFunc(w, r, db)
	})
	http.ListenAndServe(":8080", nil)
}

func handlerFunc(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	cotacao, err := getCotacao(ctx, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = insertCotacao(ctx, db, cotacao.USDBRL.Bid, "dolar")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := Cotacao{Cotacao: cotacao.USDBRL.Bid}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getCotacao(ctx context.Context, w http.ResponseWriter) (Resp, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return Resp{}, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Resp{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Resp{}, err
	}

	var v Resp
	err = json.Unmarshal(body, &v)
	if err != nil {
		return Resp{}, err
	}

	return v, nil
}

func createTable(db *sql.DB) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS cotacao_tb (id INTEGER PRIMARY KEY, cotacao TEXT, tipo TEXT)")
	if err != nil {
		log.Println(err)
	}
}

func insertCotacao(ctx context.Context, db *sql.DB, cotacao string, tipo string) error {
	stmt, err := db.PrepareContext(ctx, "INSERT INTO cotacao_tb(cotacao, tipo) values(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, cotacao, tipo)
	if err != nil {
		return err
	}

	return nil
}
