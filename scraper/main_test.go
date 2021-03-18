package main

import (
	"fmt"
	"strings"
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
	yesterdayString := strings.ToLower(fmt.Sprintf("%s_%02d_%d", yesterday.Month(), yesterday.Day(), yesterday.Year()))
	threads := []string{"/r/wallstreetbets/comments/lxi05e/daily_discussion_thread_for_" + yesterdayString + "/"}
	link := getLink(threads)
	if link != "lxi05e" {
		t.Errorf("link lxi05e was not found")
	}
}

// TODO countTickerMentions/scanComments
