package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/struqt/x/logging"
	"github.com/struqt/x/txn"

	"examples/sqlc/pg/tutorial"
)

var log = logging.NewLogger("")

func main() {
	defer os.Exit(0)
	defer log.Info("Process is ending ...")
	ctx, cancel := context.WithCancel(context.Background())
	defer log.Info("Context is cancelled")
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		tick(ctx, ticker)
	}(&wg)
	wg.Wait()
}

func tick(ctx context.Context, ticker *time.Ticker) {
	dsn := fmt.Sprintf("sslmode=disable dbname=example host=%s user=example password=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PASSWORD"))
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Error(err, "")
	}
	defer func() {
		_ = db.Close()
	}()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(600 * time.Second)
	var prepared *tutorial.Queries
	defer func() {
		if prepared != nil {
			_ = prepared.Close()
		}
	}()
	var count = 0
	for {
		if count >= 3 {
			return
		}
		select {
		case <-ctx.Done():
			log.Info("Demo Ticker is stopping ...")
			return
		case <-ticker.C:
			go func() {
				defer func() { count++ }()
				if prepared == nil {
					log.Info("Preparing ...")
					t0 := time.Now()
					prepared, err = tutorial.Prepare(ctx, db)
					if err != nil {
						log.Error(err, "failed to prepare transaction")
						return
					}
					log.Info("Prepared", "t0", t0, "t1", time.Now())
				}
				if result, err := txn.SqlExecute(ctx, db, push(prepared), PushAuthorDo); err != nil {
					log.Error(err, "")
					if prepared != nil {
						_ = prepared.Close()
						prepared = nil
					}
				} else {
					log.V(1).Info("", "title", result.Title(), "inserted", result.inserted)
				}
				if result, err := txn.SqlExecute(ctx, db, fetch(prepared), FetchLastAuthorDo); err != nil {
					log.Error(err, "")
					if prepared != nil {
						_ = prepared.Close()
						prepared = nil
					}
				} else {
					log.V(1).Info("", "title", result.Title(), "id", result.id)
				}
			}()
		}
	}
}

func fetch(query *tutorial.Queries) *FetchLastAuthorDoer {
	do := &FetchLastAuthorDoer{}
	do.query = query
	do.SetRethrowPanic(false)
	do.SetTitle("Txn.FetchLastAuthor")
	do.SetTimeout(200 * time.Millisecond)
	do.SetOptions(&sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	})
	return do
}

func push(query *tutorial.Queries) *PushAuthorDoer {
	do := &PushAuthorDoer{
		insert: tutorial.CreateAuthorParams{
			Name: "Brian Kernighan",
			Bio: sql.NullString{
				Valid:  true,
				String: "Co-author of The C Programming Language",
			},
		},
	}
	do.query = query
	do.SetRethrowPanic(false)
	do.SetTitle("Txn.FetchLastAuthor")
	do.SetTimeout(250 * time.Millisecond)
	do.SetOptions(&sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	})
	return do
}
