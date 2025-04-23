package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"
	"go.etcd.io/bbolt"
)

func main() {
	os.Exit(Main())
}

func Main() int {

	logger := slog.Default()

	var dbFlag = flag.String("db", "./db", "file where the db lives")
	var rpidFlag = flag.String("rp-id", "localhost", "relaying party id")
	var rpOriginsFlag = flag.String("rp-origins", "http://localhost:8585", "relaying party origins (comma separated, no spaces)")
	var listenAddr = flag.String("addr", ":8585", "listening addr")

	flag.Parse()

	db, err := bbolt.Open(*dbFlag, 0600, nil)
	if err != nil {
		logger.Error("opening db", "err", err)
		return 1
	}

	wconfig := &webauthn.Config{
		RPDisplayName: "Banner's Calm",
		RPID:          *rpidFlag,
		RPOrigins:     strings.Split(*rpOriginsFlag, ","),
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		logger.Error("setting up webauthn", "err", err)
		panic(err)
	}

	h := NewActivityHandler(db, webAuthn, logger)

	server := &http.Server{
		Addr:    *listenAddr,
		Handler: h,
	}

	logger.Info("listening", "addr", *listenAddr)
	err = server.ListenAndServe()
	if err != nil {
		logger.Error("serving", "err", err)
		return 1
	}

	return 0
}
