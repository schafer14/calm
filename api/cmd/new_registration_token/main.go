package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"timeapi/data"

	"go.etcd.io/bbolt"
)

func main() {

	logger := slog.Default()

	if len(os.Args) < 2 {
		fmt.Println("Username required")
		os.Exit(1)
	}

	var dbFlag = flag.String("db", "./db", "file where the db lives")

	db, err := bbolt.Open(*dbFlag, 0600, nil)
	if err != nil {
		logger.Error("opening db", "err", err)
		os.Exit(1)
	}

	stores := data.NewStores(db)
	username := os.Args[1]

	token := rand.Text()
	err = stores.CreateRegistrationToken(token, username)
	if err != nil {
		logger.Error("creating registration token", "err", err)
		os.Exit(1)
	}

	fmt.Println(username)
	fmt.Println(token)
}
