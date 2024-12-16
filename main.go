package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"sync"

	"golang.org/x/net/context"
)

var (
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	disableBan bool
)

func usage() {
	println(`Usage: reverse_proxy [options] [command] `)
}

func main() {

	disableBan := flag.Bool("disable-ban", false, "Disable the ban functionality")

	flag.Usage = usage
	flag.Parse()
	log.Print("starting")
	// Setup context
	ctx, cancel = context.WithCancel(context.Background())
	log.Print("starting")

	wg.Add(1)
	go runCmd(ctx, cancel, flag.Arg(0), flag.Args()[1:]...)

	backendURLStr := os.Getenv("BACKEND_URL")
	if backendURLStr == "" {
		backendURLStr = "http://localhost:8080"
	}

	backendURL, err := url.Parse(backendURLStr)
	if err != nil {
		log.Fatalf("Failed to parse backend URL: %v", err)
	}
	serve(backendURL, *disableBan)

	wg.Wait()
}
