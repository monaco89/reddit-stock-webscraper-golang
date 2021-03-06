package main

import (
	"fmt"
	"testing"
	"time"
)

func TestContains(t *testing.T) {
	word := "SPY"
	tickers := []string{"ABNB", "SPY"}
	contains := Contains(tickers, word)
	if contains != true {
		t.Errorf("%s did not contain %s", word, word)
	}
}

func TestGetLink(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayString := fmt.Sprintf("%s %02d, %d", yesterday.Month(), yesterday.Day(), yesterday.Year())
	threads := []Thread{{Title: "Daily Discussion Thread for " + yesterdayString, URL: "/r/wallstreetbets/comments/lxi05e/daily_discussion_thread_for_march_04_2021/"}}
	link := getLink(threads)
	if link != "lxi05e" {
		t.Errorf("link lxi05e was not found")
	}
}

// TODO countTickerMentions/scanComments
