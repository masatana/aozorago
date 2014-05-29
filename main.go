package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"code.google.com/p/go.net/html"
)

func contains(stringArray []string, a string) bool {
	for _, v := range stringArray {
		if v == a {
			return true
		}
	}
	return false
}

func retrieveAllCardPages(urls []string, urlch chan string, finch chan bool) {
	for _, url := range urls {
		if len(url) == 0 {
			continue
		}
		resp, err := http.Get(url)
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()
		d := html.NewTokenizer(resp.Body)
		for {
			tokenType := d.Next()
			if tokenType == html.ErrorToken {
				time.Sleep(time.Second * 1)
				break
			}
			token := d.Token()
			switch tokenType {
			case html.StartTagToken:
				switch token.Data {
				case "a":
					for _, v := range token.Attr {
						if v.Key == "href" && strings.HasPrefix(v.Val, "../cards") {
							urlch <- "http://www.aozora.gr.jp" + strings.Trim(v.Val, "..")
							break
						}
					}
				}
			}
		}
	}
	finch <- true
}

func retrieveAllIndexUrls(urls []string, urlch chan string, finch chan bool) {
	for _, url := range urls {
		if len(url) == 0 {
			// TODO:Should check in retrieveFirstIndexUrls
			continue
		}
		resp, err := http.Get(url)
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()
		d := html.NewTokenizer(resp.Body)
		for {
			tokenType := d.Next()
			if tokenType == html.ErrorToken {
				time.Sleep(time.Second * 1)
				break
			}
			token := d.Token()
			switch tokenType {
			case html.StartTagToken:
				switch token.Data {
				case "a":
					for _, v := range token.Attr {
						if v.Key == "href" && strings.HasPrefix(v.Val, "sakuhin") {
							urlch <- "http://www.aozora.gr.jp/index_pages/" + v.Val
							break
						}
					}
				}
			}
		}
	}
	finch <- true
}

func retrieveFirstIndexUrls(r io.Reader, urlch chan string, finch chan bool) {
	insideSakuhinListTable := false
	d := html.NewTokenizer(r)
	for {
		// token type
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			finch <- true
			break
		}
		token := d.Token()
		switch tokenType {
		case html.StartTagToken:
			switch token.Data {
			case "table":
				for _, v := range token.Attr {
					if v.Val == "作品リスト" {
						insideSakuhinListTable = true
					}
				}
			case "a":
				if !insideSakuhinListTable {
					continue
				}
				for _, v := range token.Attr {
					if v.Key == "href" {
						urlch <- "http://www.aozora.gr.jp/" + v.Val
					}
				}
			}
		case html.EndTagToken:
			if token.Data == "table" && insideSakuhinListTable {
				insideSakuhinListTable = false
			}
		}
	}
	return
}

func main() {
	urlch := make(chan string)
	finch := make(chan bool)
	urls := make([]string, 40)
	resp, err := http.Get("http://www.aozora.gr.jp/")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	go retrieveFirstIndexUrls(resp.Body, urlch, finch)
FIRSTINDEXLOOP:
	for {
		select {
		case url := <-urlch:
			urls = append(urls, url)
		case <-finch:
			break FIRSTINDEXLOOP
		}
	}
	go retrieveAllIndexUrls(urls, urlch, finch)
ALLINDEXLOOP:
	for {
		select {
		case url := <-urlch:
			urls = append(urls, url)
		case <-finch:
			break ALLINDEXLOOP
		}
	}
	allIndexUrls := make([]string, len(urls))
	for _, url := range urls {
		if len(url) == 0 {
			continue
		}
		if !contains(allIndexUrls, url) {
			allIndexUrls = append(allIndexUrls, url)
		}
	}
	go retrieveAllCardPages(allIndexUrls, urlch, finch)
ALLCARDPAGES:
	for {
		select {
		case url := <-urlch:
			fmt.Println(url)
		case <-finch:
			break ALLCARDPAGES
		}
	}
	return
}
