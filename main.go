// Runnable break-point reactor for the Live Tennis API live feed.
//
// It opens the ULTRA WebSocket feed with signals=["break_point"], routes every
// frame to a Strategy, and logs the paper action the strategy would take. It
// places NO real bets — see strategy.go.
//
//	cp .env.example .env      # then put your ULTRA key in .env
//	go run .                  # live WebSocket stream
//	go run . -rest            # REST example: list live matches
//
// The break-point feed and the WebSocket surface are ULTRA-tier only.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
)

func main() {
	os.Exit(run())
}

func run() int {
	restMode := flag.Bool("rest", false, "run the REST example (list live matches) instead of the live stream")
	flag.Parse()

	log.SetFlags(log.Ltime)
	loadDotEnv(".env")

	key := strings.TrimSpace(os.Getenv("LIVETENNISAPI_KEY"))
	if key == "" {
		log.Println("set LIVETENNISAPI_KEY (see .env.example) — the break-point feed needs an ULTRA key")
		return 2
	}

	// Cancel cleanly on Ctrl-C so the socket closes and go run exits 0.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *restMode {
		if err := runREST(ctx, key); err != nil {
			log.Println(err)
			return 1
		}
		return 0
	}

	strategy := NewStrategy()
	log.Println("connecting to the live feed with signals=[break_point] …")

	if err := stream(ctx, key, strategy); err != nil {
		var fe *feedError
		if errors.As(err, &fe) {
			switch fe.code {
			case "upgrade_required":
				log.Println("this key is not on ULTRA — the WebSocket + break-point feed require the ULTRA tier")
				return 3
			case "unauthorized":
				log.Println("the API rejected this key; check LIVETENNISAPI_KEY")
				return 3
			}
		}
		log.Println(err)
		return 1
	}
	return 0
}
