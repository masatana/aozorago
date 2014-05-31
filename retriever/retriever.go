package retriever

import (
	"io"
	"io/ioutil"
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
	connector   string
}

func NewCard() *Card {
	return &Card{connector: "_"}
}

func ConcatSpace(s string, connector string) string {
	return strings.Join(strings.Fields(s), connector)
}

func Contains(stringArray []string, a string) bool {
	for _, v := range stringArray {
		if v == a {
			return true
		}
	}
	return false
}

func (c *Card) Save(dataRootPath string) error {
	dstPath := path.Join(dataRootPath, c.author)
	err := os.MkdirAll(dstPath, 0777)
	if err != nil {
		return err
	}
	resp, err := http.Get(c.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fileContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	ioutil.WriteFile(path.Join(dstPath, c.sakuhinName+".zip"), fileContent, 0777)
	log.Printf("%s was downloaded and saved to %s\n", c.sakuhinName, dstPath)
	return nil
}

func RetrieveCards(urls []string, cardch chan Card, finch chan bool) {
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
		card := NewCard()
		isSakuhinName := false
		isAuthorName := false
		isTd := false
		isA := false
		for {
			tokenType := d.Next()
			if tokenType == html.ErrorToken {
				cardch <- *card
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
				switch {
				case isTd && strings.Contains(token.String(), "著者名："):
					isAuthorName = true
				case isTd && strings.Contains(token.String(), "作品名："):
					isSakuhinName = true
				case isA && isAuthorName && len(token.String()) != 0:
					card.author = ConcatSpace(token.String(), card.connector)
					isAuthorName = false
				case isSakuhinName && len(token.String()) != 0:
					card.sakuhinName = ConcatSpace(token.String(), card.connector)
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

func RetrieveCardPages(urls []string, urlch chan string, finch chan bool) {
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

func RetrieveAllIndexUrls(urls []string, urlch chan string, finch chan bool) {
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

func RetrieveFirstIndexUrls(r io.Reader, urlch chan string, finch chan bool) {
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
