package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
)

// Thread - Reddit thread data
type Thread struct {
	Title string
	URL   string
}

// Comment - comment data json response
type Comment struct {
	Body string `json:"body"`
}

// Response - pushshift json response
type Response struct {
	Data []string `json:"data"`
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

// init is invoked before main()
func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
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
				log.Println(err)
			}

			// log.Println(yesterday, date)
			if yesterday == date {
				log.Println("yesterday", threadDate)
			}

		}
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
	// Get the environment variable
	s3BucketName, exists := os.LookupEnv("S3_FILES_BUCKET")

	if exists {
		// Open the file for use
		file, err := os.Create(fileName)
		if err != nil {
			fmt.Println(err)
		}
		defer file.Close()

		downloader := s3manager.NewDownloader(s)
		_, errDownload := downloader.Download(file,
			&s3.GetObjectInput{
				Bucket: aws.String(s3BucketName),
				Key:    aws.String(fileName),
			})

		if errDownload != nil {
			fmt.Println(err)
		}
	}

	return errors.New("Can't get env variable")
}

func grabStockList() []string {
	fileName := "/tmp/tickers.csv"
	var tickers []string
	// For API, https://dumbstockapi.com/stock?format=tickers-only&exchange=NYSE
	// https://dumbstockapi.com/stock?format=tickers-only&exchange=NASDAQ

	// Check if file already exists
	_, err := os.Stat(fileName)
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
	rows, err := csv.NewReader(file).ReadAll()
	file.Close()
	if err != nil {
		log.Fatal(err)
	}

	// We only want the first column (tickers)
	for _, line := range rows[1:] {
		tickers = append(tickers, line[0])
	}

	return tickers
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
	replacer := strings.NewReplacer(",", "", ".", "", ";", "")
	// Loop through each comment body field
	for _, comment := range commentsText {
		text := replacer.Replace(comment.Body)
		words := strings.Fields(text)
		// Loop through each word in body
		for _, word := range words {
			// Scan for each stock ticker in comment body then add to Stocks map
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
	// Can only query 500 ids at a time
	// Loop through array 500 each
	i := 0
	// ! Testing, should be 0
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

func sortMap(unsortedMap map[string]StockMentions) map[string]StockMentions {
	var sortedMap map[string]StockMentions
	var keys []string
	for k := range unsortedMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// To perform the opertion you want
	for _, k := range keys {
		log.Println("Key:", k, "Value:", unsortedMap[k])
		// TODO assign sortedMap
	}

	return sortedMap
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

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, fileDir string) error {
	// Get the environment variable
	s3BucketName, exists := os.LookupEnv("S3_FILES_BUCKET")

	if exists {
		// Open the file for use
		file, err := os.Open(fileDir)
		if err != nil {
			return err
		}
		defer file.Close()

		// Get file size and read the file content into a buffer
		fileInfo, _ := file.Stat()
		var size int64 = fileInfo.Size()
		buffer := make([]byte, size)
		file.Read(buffer)

		// Config settings: this is where you choose the bucket, filename, content-type etc.
		// of the file you're uploading.
		_, err = s3.New(s).PutObject(&s3.PutObjectInput{
			Bucket:               aws.String(s3BucketName),
			Key:                  aws.String(fileDir),
			ACL:                  aws.String("private"),
			Body:                 bytes.NewReader(buffer),
			ContentLength:        aws.Int64(size),
			ContentType:          aws.String(http.DetectContentType(buffer)),
			ContentDisposition:   aws.String("attachment"),
			ServerSideEncryption: aws.String("AES256"),
		})
		return err
	}

	return errors.New("Can't get env variable")
}

func uploadToS3() {
	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		log.Fatal(err)
	}

	// Upload
	err = AddFileToS3(s, "/tmp/redditStocks.csv")
	if err != nil {
		log.Fatal(err)
	}
}

func startTheShow() {
	// threads := grabHTML()
	log.Println("Grabbing discussion id...")
	// linkID := getLink(threads)
	// ! For testing
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
	log.Println("Writing count to CSV...")
	writeToCsv()
	log.Println("Uploading CSV to S3...")
	uploadToS3()
}

func main() {
	lambda.Start(startTheShow)
}
