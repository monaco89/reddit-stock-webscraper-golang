package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"net/http"

	"github.com/gocolly/colly"
)

type Thread struct {
	Title string
	URL   string
}

type Comment struct {
	Body string `json:"body"`
}
type Response struct {
	Data []string `json:"data"`
}

type CommentResponse struct {
	Data []Comment `json:"data"`
}

type StockMentions struct {
	// symbol   string
	Mentions int
}

var Stocks = make(map[string]StockMentions)

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func grabHTML() []Thread {
	threads := []Thread{}
	url := "https://www.reddit.com/r/wallstreetbets/search/?q=flair%3A%22Daily%20Discussion%22&restrict_sr=1&sort=new"
	c := colly.NewCollector()

	// _eYtD2XCVieq6emjKBH3m
	// We want all objects with this specific class
	c.OnHTML("._2INHSNB8V5eaWp4P0rY_mE", func(e *colly.HTMLElement) {
		discussion := Thread{}
		discussion.Title = e.Text
		discussion.URL = e.Attr("href")
		threads = append(threads, discussion)
	})

	//onHTML function allows the collector to use a callback function when the specific HTML tag is reached
	//in this case whenever our collector finds an
	//anchor tag with href it will call the anonymous function
	// specified below which will get the info from the href and append it to our slice
	// c.OnHTML("a[href]", func(e *colly.HTMLElement) {
	// 	link := e.Request.AbsoluteURL(e.Attr("href"))
	// 	if link != "" {
	// 		response = append(response, link)
	// 	}
	// })

	// c.OnScraped(func(r *colly.Response) {
	// 	log.Println("Finished. Here is your data:", threads)
	// 	// parse our response slice into JSON format
	// 	// b, err := json.Marshal(threads)
	// 	// if err != nil {
	// 	// 	log.Println("failed to serialize response:", err)
	// 	// 	return
	// 	// }
	// })

	err := c.Visit(url)
	if err != nil {
		log.Println(err)
	}

	return threads
}

func getLink(threads []Thread) string {
	// yesterday := time.Now().AddDate(0, 0, -1).Format("February 16, 2021")
	yesterday := time.Now().AddDate(0, 0, -1)
	link := ""

	for _, thread := range threads {
		/*
		   Check if it's a DD or weekend thread
		   Then split up text to only get last three parts
		   Conver to datetime then compare to yesterdays date
		   If equal, grab link from parent element
		*/
		if strings.HasPrefix(thread.Title, "Daily Discussion Thread") {
			threadTitle := strings.Split(thread.Title, " ")
			threadDate := strings.Join(threadTitle[len(threadTitle)-3:], " ")

			// TODO Format threadDate to a time format to compare with
			date, err := time.Parse("February 16, 2021", threadDate)

			if err != nil {
				fmt.Println(err)
			}

			fmt.Println(yesterday, date)
			if yesterday == date {
				fmt.Println("yesterday", threadDate)
			}

		}
	}

	return link
}

func grabCommentIds(linkID string) []string {
	resp, err := http.Get(fmt.Sprintf("https://api.pushshift.io/reddit/submission/comment_ids/%s", linkID))
	if err != nil {
		fmt.Println(err)
	}
	// Read body then convert to string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	sb := string(body)

	defer resp.Body.Close()
	var cResp Response
	// Parse the json string
	if json.Unmarshal([]byte(sb), &cResp); err != nil {
		fmt.Println(err)
	}

	return cResp.Data
}

func grabStockList() []string {
	// For API, https://dumbstockapi.com/stock?format=tickers-only&exchange=NYSE
	// https://dumbstockapi.com/stock?format=tickers-only&exchange=NASDAQ
	f, err := os.Open("tickers.csv")
	if err != nil {
		log.Fatal(err)
	}
	// Get rows from csv
	rows, err := csv.NewReader(f).ReadAll()
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	// We only want the first column (tickers)
	var tickers []string
	for _, line := range rows[1:] {
		tickers = append(tickers, line[0])
	}

	return tickers
}

func getComments(idsString string) []Comment {
	resp, err := http.Get(fmt.Sprintf("https://api.pushshift.io/reddit/comment/search?ids=%s&fields=body&size=500", idsString))
	if err != nil {
		fmt.Println(err)
	}

	// Read body then convert to string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	sb := string(body)

	defer resp.Body.Close()
	var cResp CommentResponse
	// Parse the json string
	if json.Unmarshal([]byte(sb), &cResp); err != nil {
		fmt.Println(err)
	}

	return cResp.Data
}

func countTickerMentions(commentsText []Comment, tickers []string) {
	replacer := strings.NewReplacer(",", "", ".", "", ";", "")
	// Loop through each comment body field
	for _, comment := range commentsText {
		text := replacer.Replace(comment.Body)
		words := strings.Fields(text)
		// Loop through each word in body
		// fmt.Println(words)
		for _, word := range words {
			// fmt.Println(j, " => ", word)
			// Scan for each stock ticker in comment body then add to Stocks map
			isTicker := Contains(tickers, word)

			if isTicker {
				fmt.Println("Found:", word)
				count := Stocks[word].Mentions
				mentions := StockMentions{Mentions: count + 1}
				Stocks[word] = mentions
			}
		}
	}
}

func scanComments(commentIds []string, tickers []string) {
	orgList := commentIds
	// Can only query 500 ids at a time
	// Loop through array 500 each
	i := 0
	// ! Testing
	for 35000 < len(orgList) {
		// Get first 500 ids, put in string
		// idsString := strings.Join(orgList[0:500], ",")
		// ! Testing
		idsString := strings.Join(orgList[0:15], ",")
		// Removed used ids
		orgList = orgList[i*500:]
		// Get comment text
		commentsText := getComments(idsString)
		// Count stock ticker mentions
		countTickerMentions(commentsText, tickers)
		i++
	}
}

func main() {
	// threads := grabHTML()
	log.Println("Grabbing discussion id...")
	// linkID := getLink(threads)
	linkID := "lra5cg"
	// log.Println("Link: ", linkID)
	log.Println("Grabbing comment id...")
	commentIds := grabCommentIds(linkID)
	log.Println("# of ids...", len(commentIds))
	log.Println("Grabbing stock symbols from csv...")
	tickers := grabStockList()
	// Get stocks from comments
	log.Println("Counting stock mentions...")
	scanComments(commentIds, tickers)
	log.Println(Stocks)
	// TODO Write results to csv
	// TODO Upload to S3
}
