package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"timeapi/authz"

	"go.etcd.io/bbolt"
)

func main() {

	logger := slog.Default()

	if len(os.Args) < 3 {
		fmt.Println("Username and permission required")
		os.Exit(1)
	}

	var dbFlag = flag.String("db", "./db", "file where the db lives")

	db, err := bbolt.Open(*dbFlag, 0600, nil)
	if err != nil {
		logger.Error("opening db", "err", err)
		os.Exit(1)
	}

	e := authz.NewBbolt(db)
	_, err = e.AddRoleForUser(os.Args[1], os.Args[2])
	if err != nil {
		logger.Error("creating permission", "err", err)
		os.Exit(1)
	}
}
