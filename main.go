package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"code.google.com/p/go.net/html"
)

type Card struct {
	author      string
	sakuhinName string
	url         string
}

func contains(stringArray []string, a string) bool {
	for _, v := range stringArray {
		if v == a {
			return true
		}
	}
	return false
}

func (c *Card) Save(saveRootPath string) error {
	savePath := path.Join(saveRootPath, c.author)
	err := os.MkdirAll(savePath, 0777)
	if err != nil {
		return err
	}
	fmt.Println(c)
	/*
		resp, err := http.Get(c.url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		fileContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		ioutil.WriteFile(path.Join(savePath, c.sakuhinName+".zip"), fileContent, 0777)
	*/
	return nil
}

func retrieveCards(urls []string, cardch chan Card, finch chan bool) {
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
		var card Card
		isSakuhinName := false
		isAuthorName := false
		isTd := false
		isA := false
		for {
			tokenType := d.Next()
			if tokenType == html.ErrorToken {
				cardch <- card
				time.Sleep(time.Second * 1)
				break
			}
			token := d.Token()
			switch tokenType {
			case html.StartTagToken:
				switch token.Data {
				case "a":
					isA = true
					for _, v := range token.Attr {
						if v.Key == "href" && strings.HasSuffix(v.Val, ".zip") {
							s := strings.Split(url, "/")
							card.url = strings.Join(s[:len(s)-1], "/") + v.Val[1:]
							break
						}
					}
				case "td":
					isTd = true
				}
			case html.TextToken:
				fmt.Println(token)
				switch {
				case isTd && strings.Contains(token.String(), "著者名："):
					isAuthorName = true
				case isTd && strings.Contains(token.String(), "作品名："):
					isSakuhinName = true
				case isA && isAuthorName && len(token.String()) != 0:
					card.author = token.String()
					isAuthorName = false
				case isSakuhinName && len(token.String()) != 0:
					card.sakuhinName = token.String()
					isSakuhinName = false
				}
			case html.EndTagToken:
				switch token.Data {
				case "td":
					isTd = false
				case "a":
					isA = false
				}
			}
		}
	}
	finch <- true
}

func retrieveCardPages(urls []string, urlch chan string, finch chan bool) {
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
	cardPageUrls := make([]string, 100)
	go retrieveCardPages(allIndexUrls, urlch, finch)
ALLCARDPAGES:
	for {
		select {
		case url := <-urlch:
			cardPageUrls = append(cardPageUrls, url)
		case <-finch:
			break ALLCARDPAGES
		}
	}
	cardch := make(chan Card)
	go retrieveCards(cardPageUrls, cardch, finch)
ALLCARDS:
	for {
		select {
		case card := <-cardch:
			err := card.Save("/home/masatana/aozora.gr.jp")
			if err != nil {
				log.Fatalln(err)
			}
		case <-finch:
			break ALLCARDS
		}
	}
	return
}
