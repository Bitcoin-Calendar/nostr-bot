package main

import (
	"calendar-bot/internal/api"
	"calendar-bot/internal/bot"
	"context"
	"flag"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"
)

var version = "development"

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
func execute() error {
	check := flag.Bool("check", false, "Validate key, API, formatting and relay reads without publishing")
	probe := flag.Bool("probe", false, "Check API, formatting, state and relay reads without a private key or publishing")
	preview := flag.Bool("preview", false, "Print unsigned kind 1 posts; no private key required")
	date := flag.String("date", "", "YYYY-MM-DD; only with --preview")
	ver := flag.Bool("version", false, "Print build version without reading configuration")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *ver {
		if *check || *preview || *probe || *date != "" {
			return fmt.Errorf("--version cannot be combined")
		}
		fmt.Println("bitcal-nostr build=" + version)
		return nil
	}
	modes := 0
	for _, selected := range []bool{*check, *preview, *probe} {
		if selected {
			modes++
		}
	}
	if modes > 1 {
		return fmt.Errorf("choose --check, --probe or --preview")
	}
	if *date != "" && !*preview {
		return fmt.Errorf("--date is only allowed with --preview")
	}
	c, err := bot.LoadConfig(!*preview && !*probe)
	if err != nil {
		return err
	}
	now := time.Now()
	if *date != "" {
		now, err = time.ParseInLocation("2006-01-02", *date, c.Location)
		if err != nil {
			return fmt.Errorf("invalid --date")
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	client := api.NewClient(c.APIURL, c.APIKey)
	if *check || *preview || *probe {
		events, err := client.FetchEvents(now.In(c.Location).Format("01"), now.In(c.Location).Format("02"), "en")
		if err != nil {
			return err
		}
		for _, event := range events {
			if event.Date.Format("01-02") != now.In(c.Location).Format("01-02") {
				return fmt.Errorf("API returned wrong event date")
			}
			note, err := bot.Format(c, event)
			if err != nil {
				return err
			}
			if *preview {
				fmt.Printf("--- event=%d kind=%d ---\n%s\n", event.ID, note.Kind, note.Content)
			}
		}
		fmt.Printf("API and formatting passed: date=%s events=%d\n", now.In(c.Location).Format("2006-01-02"), len(events))
		if *check || *probe {
			if *check {
				pubkey, _ := nostr.GetPublicKey(c.PrivateKey)
				npub, _ := nip19.EncodePublicKey(pubkey)
				fmt.Println("Account:", npub)
			}
			store, err := bot.OpenStore(c.StateDir)
			if err != nil {
				return err
			}
			store.Close()
			if err := bot.CheckRelays(ctx, c.Relays, os.Stdout); err != nil {
				return err
			}
			fmt.Println("CHECK PASSED: API, formatting, state lock and relay reads. No posts sent; write acceptance unverified.")
		}
		return nil
	}
	return bot.Run(ctx, c, now, client.FetchEvents, bot.PublishToRelays(c.Relays, os.Stdout), os.Stdout)
}
