package main

//go:generate ./version.sh

import (
	"log"
	"net/http"
	"time"

	"github.com/PagerDuty/godspeed"
	"github.com/codegangsta/cli"
	"github.com/mozilla-services/go-bouncer/bouncer"
	_ "github.com/mozilla-services/go-bouncer/mozlog"
)

func main() {
	app := cli.NewApp()
	app.Name = "bouncer"
	app.Action = Main
	app.Version = bouncer.Version
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "cache-time",
			Value: 60,
			Usage: "Time, in seconds, for Cache-Control max-age",
		},
		cli.StringFlag{
			Name:   "addr",
			Value:  ":8888",
			Usage:  "address on which to listen",
			EnvVar: "BOUNCER_ADDR",
		},
		cli.StringFlag{
			Name:   "db-dsn",
			Value:  "user:password@tcp(localhost:3306)/bouncer",
			Usage:  "database DSN (https://github.com/go-sql-driver/mysql#dsn-data-source-name)",
			EnvVar: "BOUNCER_DB_DSN",
		},
		cli.BoolFlag{
			Name:  "dogstatsd",
			Usage: "Enable dogstatsd metrics",
		},
	}
	app.RunAndExitOnError()
}

func Main(c *cli.Context) {
	db, err := bouncer.NewDB(c.String("db-dsn"))
	if err != nil {
		log.Fatalf("Could not open DB: %v", err)
	}
	defer db.Close()

	bouncerHandler := &BouncerHandler{
		db:        db,
		CacheTime: time.Duration(c.Int("cache-time")) * time.Second,
	}
	if c.Bool("dogstatsd") {
		gs, err := godspeed.NewDefault()
		if err != nil {
			log.Println(err)
		} else {
			defer gs.Conn.Close()
			bouncerHandler.SetGodspeed(gs, 30000)
		}
	}

	healthHandler := &HealthHandler{
		db:        db,
		CacheTime: 5 * time.Second,
	}

	mux := http.NewServeMux()

	mux.Handle("/__heartbeat__", healthHandler)
	mux.Handle("/", bouncerHandler)

	server := &http.Server{
		Addr:    c.String("addr"),
		Handler: mux,
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
