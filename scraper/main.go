package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Thread - Reddit thread data
type Thread struct {
	Title     string `json:"title"`
	Permalink string `json:"permalink"`
}

// ThreadResponse - post query response
type ThreadResponse struct {
	Data []Thread `json:"data"`
}

// Comment - comment data json response
type Comment struct {
	Body string `json:"body"`
}

// Response - pushshift json response
type Response struct {
	Data []string `json:"data"`
}

// JSONFileResponse - repsonse from s3 file
type JSONFileResponse struct {
	CikStr int    `json:"cik_str"`
	Ticker string `json:"ticker"`
	Title  string `json:"title"`
}

// CommentResponse - pushshift comment data json response
type CommentResponse struct {
	Data []Comment `json:"data"`
}

// StockMentions - data structure for CSV
type StockMentions struct {
	// symbol   string
	Mentions int
}

// Stocks - global variable stock ticker counter
var Stocks = make(map[string]StockMentions)

// Contains finds string in array of strings
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// TODO Have option to run scraper
// For Web scraper
// func grabHTML() []string {
// 	// threads := []Thread{}
// 	var threads []string
// 	url := "https://www.reddit.com/r/wallstreetbets/search/?q=flair%3A%22Daily%20Discussion%22&restrict_sr=1&sort=new"

// 	// create chrome instance
// 	ctx, cancel := chromedp.NewContext(
// 		context.Background(),
// 		chromedp.WithLogf(log.Printf),
// 	)
// 	defer cancel()

// 	// create a timeout
// 	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
// 	defer cancel()

// 	// navigate
// 	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
// 		log.Printf("could not navigate to wallstreetbets: %v", err)
// 	}

// 	// get project link text
// 	var nodes []*cdp.Node
// 	if err := chromedp.Run(ctx, chromedp.Nodes("._2INHSNB8V5eaWp4P0rY_mE", &nodes, chromedp.ByQueryAll)); err != nil {
// 		log.Printf("could not get nodes: %v", err)
// 	}

// 	// process data
// 	for i := 0; i < len(nodes); i++ {
// 		if nodes[i].AttributeValue("href") != "" {
// 			// var title string
// 			// if err := chromedp.Run(ctx, chromedp.Text(nodes[i].NodeValue, &title, chromedp.NodeVisible, chromedp.ByQuery)); err != nil {
// 			// 	log.Printf("could not get title: %v", err)
// 			// }

// 			// res = append(res, Thread{
// 			// 	URL:   nodes[i].AttributeValue("href"),
// 			// 	Title: title,
// 			// })

// 			// Just append href of threads
// 			threads = append(threads, nodes[i].AttributeValue("href"))
// 		}
// 	}

// 	return threads
// }

// func getLinkFromScraper(threads []string) string {
// 	yesterday := time.Now().AddDate(0, 0, -1)
// 	yesterdayString := strings.ToLower(fmt.Sprintf("%s %02d %d", yesterday.Month(), yesterday.Day(), yesterday.Year()))
// 	link := ""

// 	for _, thread := range threads {
// 		/*
// 		   Check if it's a DD or weekend thread
// 		   Then split up text to only get last three parts
// 		   Check if yesterday string equals thread date
// 		   If equal, grab link from parent element
// 		*/

// 		// e.g. daily_discussion_thread_for_march_16_2021
// 		threadURL := strings.Split(thread, "/")
// 		threadTitle := threadURL[len(threadURL)-2]

// 		if strings.HasPrefix(threadTitle, "daily_discussion_thread_for") {
// 			threadDateMap := strings.Split(threadURL[len(threadURL)-2], "_")
// 			threadDate := strings.Join(threadDateMap[len(threadDateMap)-3:], " ")

// 			if yesterdayString == threadDate {
// 				threadURL := strings.Split(thread, "/")
// 				link = threadURL[4]
// 				break
// 			}
// 		}
// 	}

// 	if link == "" {
// 		panic("Couldn't get a link id")
// 	}

// 	return link
// }

func grabThreads() []Thread {
	resp, err := http.Get("https://api.pushshift.io/reddit/search/submission/?q=daily%20discussion&after=48h&subreddit=wallstreetbets&sort=desc&sort_type=num_comments&size=5")
	if err != nil {
		log.Println(err)
	}
	// Read body then convert to string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	sb := string(body)

	defer resp.Body.Close()
	var cResp ThreadResponse
	// Parse the json string
	if json.Unmarshal([]byte(sb), &cResp); err != nil {
		log.Println(err)
	}

	return cResp.Data
}

func getLinkFromAPI(threads []Thread) string {
	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayString := strings.ToLower(fmt.Sprintf("%s %02d %d", yesterday.Month(), yesterday.Day(), yesterday.Year()))
	link := ""

	for _, thread := range threads {
		/*
		   Check if it's a DD or weekend thread
		   Then split up text to only get last three parts
		   Check if yesterday string equals thread date
		   If equal, grab link from parent element
		*/

		// e.g. daily_discussion_thread_for_march_16_2021
		threadURL := strings.Split(thread.Permalink, "/")
		threadTitle := threadURL[len(threadURL)-2]

		if strings.HasPrefix(threadTitle, "daily_discussion_thread_for") {
			threadDateMap := strings.Split(threadURL[len(threadURL)-2], "_")
			threadDate := strings.Join(threadDateMap[len(threadDateMap)-3:], " ")

			if yesterdayString == threadDate {
				threadURL := strings.Split(thread.Permalink, "/")
				link = threadURL[4]
				break
			}
		}
	}

	if link == "" {
		panic("Couldn't get a link id")
	}

	return link
}

func grabCommentIds(linkID string) []string {
	resp, err := http.Get("https://api.pushshift.io/reddit/submission/comment_ids/" + linkID)
	if err != nil {
		log.Println(err)
	}
	// Read body then convert to string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	sb := string(body)

	defer resp.Body.Close()
	var cResp Response
	// Parse the json string
	if json.Unmarshal([]byte(sb), &cResp); err != nil {
		log.Println(err)
	}

	return cResp.Data
}

func getFileFromS3(s *session.Session, fileName string) error {
	// Open the file for use
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(s)
	_, errDownload := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("S3_FILES_BUCKET")),
			Key:    aws.String(fileName),
		})

	return errDownload
}

func grabStockList() []string {
	fileName := "tickers.json"
	var tickers []string

	// Check if file already exists
	_, err := os.Stat("/tmp/" + fileName)
	if os.IsNotExist(err) {
		// Create a single AWS session
		s, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
		if err != nil {
			log.Fatal(err)
		}

		// Get file from s3 bucket
		err = getFileFromS3(s, fileName)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Open the file downloaded from s3
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}

	// Get rows from csv
	// rows, err := csv.NewReader(file).ReadAll()
	// file.Close()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Println("rows length", len(rows))

	// // We only want the first column (tickers)
	// for _, line := range rows[1:] {
	// 	tickers = append(tickers, line[0])
	// }

	// Read file body then convert to string
	body, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
	}
	sb := string(body)

	defer file.Close()
	var cResp map[string]JSONFileResponse
	// Parse the json string
	if json.Unmarshal([]byte(sb), &cResp); err != nil {
		log.Println(err)
	}

	for _, data := range cResp {
		tickers = append(tickers, data.Ticker)
	}

	return tickers
}

func fetchStockList() []string {
	resp, err := http.Get("https://dumbstockapi.com/stock?format=tickers-only&exchange=NYSE,NASDAQ")
	if err != nil {
		log.Println(err)
	}
	// Read body then convert to string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	sb := string(body)

	defer resp.Body.Close()
	var cResp []string
	// Parse the json string
	if json.Unmarshal([]byte(sb), &cResp); err != nil {
		log.Println(err)
	}

	return cResp
}

func getComments(idsString string) []Comment {
	resp, err := http.Get("https://api.pushshift.io/reddit/comment/search?ids=" + idsString + "&fields=body&size=500")
	if err != nil {
		log.Println(err)
	}

	// Read body then convert to string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	sb := string(body)

	defer resp.Body.Close()
	var cResp CommentResponse
	// Parse the json string
	if json.Unmarshal([]byte(sb), &cResp); err != nil {
		log.Println(err)
	}

	return cResp.Data
}

func countTickerMentions(commentsText []Comment, tickers []string) {
	// Remove characters from words
	replacer := strings.NewReplacer(",", "", ".", "", ";", "")
	// Loop through each comment body field
	for _, comment := range commentsText {
		text := replacer.Replace(comment.Body)
		words := strings.Fields(text)
		// Loop through each word in body
		for _, word := range words {
			// Scan for each stock ticker in a single word then add to Stocks map
			isTicker := Contains(tickers, word)

			if isTicker {
				// log.Println("Found:", word)
				count := Stocks[word].Mentions
				mentions := StockMentions{Mentions: count + 1}
				Stocks[word] = mentions
			}
		}
	}
}

func scanComments(commentIds []string, tickers []string) {
	orgList := commentIds
	// Set max limit of ids (might reach 15 min timeout)
	if len(orgList) > 40000 {
		orgList = orgList[0:40000]
	}
	// Can only query 500 ids at a time
	// Loop through array 500 each
	i := 0
	for 0 < len(orgList) {
		log.Println("comment ids left...", len(orgList))
		// Get first 500 ids, put in string
		idsString := strings.Join(orgList[0:500], ",")
		// Removed used ids
		if len(orgList) < 500 {
			orgList = orgList[0:0]
		} else {
			orgList = orgList[500:]
		}
		// Get comment text
		commentsText := getComments(idsString)
		// Count stock ticker mentions
		countTickerMentions(commentsText, tickers)
		i += i
	}
}

func writeToCsv() {
	// Write tmp file
	file, err := os.Create("/tmp/redditStocks.csv")
	if err != nil {
		log.Println(err)
	}

	w := csv.NewWriter(file)
	// TODO Invoke sorted map method to sort them count
	// Loop through global variable and write
	for key, stockset := range Stocks {
		err := w.Write([]string{fmt.Sprintf("%v", key), fmt.Sprintf("%v", stockset.Mentions)})
		if err != nil {
			log.Println(err)
		}
	}

	w.Flush()
}

// AddFileToS3 will upload a single file to S3
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, fileName string) error {
	// Open the file for use
	file, err := os.Open("/tmp/" + fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(os.Getenv("S3_FILES_BUCKET")),
		Key:                  aws.String("/reddit_stocks/" + fileName),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})

	return err
}

func uploadToS3(linkID string) {
	fileName := "discussion_" + linkID + ".csv"
	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		log.Fatal(err)
	}

	err = AddFileToS3(s, fileName)
	if err != nil {
		log.Fatal(err)
	}
}

func startTheShow() {
	threads := grabThreads()
	log.Println(threads)
	log.Println("Grabbing discussion id...")
	linkID := getLinkFromAPI(threads)
	log.Println(linkID)
	log.Println("Grabbing comment id...")
	commentIds := grabCommentIds(linkID)
	log.Println("# of ids...", len(commentIds))
	log.Println("Grabbing stock symbols from csv...")
	// tickers := fetchStockList()
	tickers := grabStockList()
	log.Println("Counting stock mentions...")
	scanComments(commentIds, tickers)
	log.Println(Stocks)
	log.Println("Writing count to CSV...")
	writeToCsv()
	log.Println("Uploading CSV to S3...")
	uploadToS3(linkID)
}

func main() {
	env := os.Getenv("ENV")
	if env == "local" {
		startTheShow()
	} else {
		lambda.Start(startTheShow)
	}
}
