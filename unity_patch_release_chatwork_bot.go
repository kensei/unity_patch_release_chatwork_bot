package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"io"
	"os"
	"strings"
	"github.com/PuerkitoBio/goquery"
	rss "github.com/jteeuwen/go-pkg-rss"
)

const fileName = "readed.txt"

type Rss struct {
	title string
	description string
}

var chatworkTokenPtr = flag.String("t", "", "chatwork api token")
var chatworkRoomIdPtr = flag.String("r", "", "chatwork room id")
var readedItem = []string{}
var newReadedItem = []string{}

func main() {
	if parseOpt() == false {
		return
	}
	setReaded()
	pollFeed("http://unity3d.com/unity/qa/patch-releases/latest.xml", 5)
	writeReaded()
}

func parseOpt() bool {
	flag.Parse()

	if *chatworkTokenPtr == "" {
		fmt.Println("chatwork token is required.\nuseage --help")
		return false
	}
	if *chatworkRoomIdPtr == "" {
		fmt.Println("chatwork room id is required.\nuseage --help")
		return false
	}
	return true
}

func setReaded() {
	var reader *bufio.Reader
	var line []byte
	var err error

	readFile, _ := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0600)
	reader = bufio.NewReader(readFile)

	for {
		if err == io.EOF {
			return
		}
		if line, err = reader.ReadBytes('\n'); err == nil {
			readedItem = append(readedItem, strings.Trim(string(line[:]), "\n"))
		}
	}
}

func writeReaded() {
	var writer *bufio.Writer

	if writeFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600); err == nil {
		writer = bufio.NewWriter(writeFile)

		for i := range newReadedItem {
			// fmt.Printf("added: %s\n", newReadedItem[i])
			writer.Write([]byte(newReadedItem[i] + "\n"))
		}
		writer.Flush()
	} else {
		fmt.Fprintf(os.Stderr, "[e] %s\n", err)
	}
}

func pollFeed(uri string, timeout int) {
	feed := rss.New(timeout, true, nil, itemHandler)

	if err := feed.Fetch(uri, nil); err != nil {
		fmt.Fprintf(os.Stderr, "[e] %s: %s", uri, err)
		return
	}
}

func contains(target string) bool {
	for _, item := range readedItem {
		if item == strings.Trim(target, "") {
			return true
		}
	}
	return false
}

func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
	result := []Rss{}
	for _, item := range newitems {
		if !contains(item.Title) {
			r := strings.NewReader(item.Description)
			resultBody := parseItem(r)
			newReadedItem = append(newReadedItem, item.Title)
			// fmt.Printf("%s\n", item.Title)
			// fmt.Printf("%s\n", resultBody)
			result = append(result, Rss{item.Title, resultBody})
		}
	}
	postChatrowk(result)
}

func parseItem(r io.Reader) string {
	result := ""
	if doc, err := goquery.NewDocumentFromReader(r); err != nil {
		fmt.Fprintf(os.Stderr, "[e] %s\n", err)
	} else {
		doc.Find("ul").Each(func(_ int, s *goquery.Selection) {
			result += s.Text()
		})
	}
	return result
}

func postChatrowk(data []Rss) {
	postString := ""
	for _, rss := range data {
		postString += "[info][title]" + rss.title + "[/title]" + rss.description + "[/info]"
	}
	fmt.Printf("%s\n", postString)
	postUrl := strings.Join([]string{"https://api.chatwork.com/v1/rooms/", *chatworkRoomIdPtr, "/messages"}, "")
	values := url.Values{}
	values.Add("body", postString)
	if req, reqErr := http.NewRequest("POST", postUrl, strings.NewReader(values.Encode())); reqErr == nil {
		req.Header.Add("X-ChatWorkToken", *chatworkTokenPtr)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		client := &http.Client{}
		if resp, respErr := client.Do(req); respErr != nil {
			fmt.Fprintf(os.Stderr, "[e] %s %s", respErr, resp)
		}
		defer req.Body.Close()
	} else {
		fmt.Fprintf(os.Stderr, "[e] %s", reqErr)
	}
}
