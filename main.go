package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/net/context"
)

var (
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
)

func main() {

	disableBan := flag.Bool("disable-ban", false, "Disable the ban functionality just to audit the behaviour")
	hit404threshold := flag.Int("hit-404-threshold", 50, "Threshold for 404 hits before taking action")
	banDurantionInMinutes := flag.Int("ban-duration-in-minutes", 1, "Threshold for 404 hits before taking action")
	modifyHost := flag.Bool("modify-host", false, "modify the Host in url based on BACKEND_URL")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: %s\n", os.Args[0], "followed by some option flags and the command to launch/proxy")
		flag.PrintDefaults() // Print the default flag descriptions
	}

	if len(os.Args) == 1 {
		fmt.Println("No flags provided. See usage below:")
		flag.Usage()
		os.Exit(1)
	}

	flag.Parse()
	log.Print("starting")
	// Setup context
	ctx, cancel = context.WithCancel(context.Background())
	log.Print("starting")

	wg.Add(1)
	go runCmd(ctx, cancel, flag.Arg(0), flag.Args()[1:]...)

	adminPassword := os.Getenv("BANME_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = uuid.NewString()
		fmt.Println("to access /__banme/ console and api use generated a default password for admin ", adminPassword)
	}

	backendURLStr := os.Getenv("BANME_BACKEND_URL")
	if backendURLStr == "" {
		backendURLStr = "http://localhost:8080"
	}

	backendURL, err := url.Parse(backendURLStr)
	if err != nil {
		log.Fatalf("Failed to parse backend URL: %v", err)
	}
	serve(backendURL, *disableBan, *hit404threshold, *banDurantionInMinutes, *modifyHost, adminPassword)

	wg.Wait()
}
