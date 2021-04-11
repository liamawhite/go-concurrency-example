package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var urls = []string{
	"http://golang.org/",
	"http://blog.golang.org/",
}

type Page struct {
	address string
	content io.ReadCloser
}

func main() {
	// Pending must be buffered because the pipeline is circular and there are multiple links per page.
	// This is preferred over adding to pending concurrently as it sets an upper bound on memory usage.
	// There is an assumption here that one page will not contain more than 10000 links.
	pending := make(chan string, 10000)
	retrieved := make(chan Page)
	record := VisitedPageTracker(pending)

	// Send initial urls to be recorded
	go func() { record <- urls }()

	go RetrieveContent(pending, retrieved)
	go ParseForLinks(retrieved, record)

	// Run the code for 5 seconds.
	// Again, I'm lazy and this is a contrived example.
	time.Sleep(time.Second * 5)
}

func RetrieveContent(urls <-chan string, retrieved chan<- Page) {
	for url := range urls {
		if resp, err := http.Get(url); err == nil {
			retrieved <- Page{address: url, content: resp.Body}
		}
	}
}

func ParseForLinks(retrieved <-chan Page, record chan<- []string) {
	for page := range retrieved {
		record <- parse(page)
	}
}

func VisitedPageTracker(pending chan<- string) chan<- []string {
	record := make(chan []string)
	visitedPages := map[string]bool{}
	go func() {
		for pages := range record {
			for _, page := range pages {
				if !visitedPages[page] {
					fmt.Println("recording page", page)
					visitedPages[page] = true
					pending <- page
				}
			}
		}
	}()
	return record
}

func parse(page Page) []string {
	// use goquery because I'm lazy!
	doc, _ := goquery.NewDocumentFromReader(page.content)
	page.content.Close()
	res := make([]string, 0)
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		// This logic isn't quite correct, but the point is to demonstrate concurrency not the structure of URIs
		parsed, _ := url.Parse(link)
		switch {
		case parsed.IsAbs():
			// do nothing
		case strings.HasPrefix(link, "/"):
			link = page.address + link[1:]
		default:
			link = page.address + link
		}
		res = append(res, link)
	})
	return res
}
