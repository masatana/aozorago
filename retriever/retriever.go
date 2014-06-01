package retriever

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"code.google.com/p/go.net/html"
)

// Card stores card page's information.
// (e.g. http://www.aozora.gr.jp/cards/000020/card2569.html
type Card struct {
	author      string
	sakuhinName string
	U           *url.URL
	connector   string
}

// NewCard is a constructor of Card.
// Define the default connector.
func NewCard() *Card {
	return &Card{connector: "_"}
}

// ConcatSpace concatenates a space-divided string with a connector.
func ConcatSpace(s string, connector string) string {
	return strings.Join(strings.Fields(s), connector)
}

// Contains checks if a string exists in an array of strings
func Contains(stringArray []string, a string) bool {
	for _, v := range stringArray {
		if v == a {
			return true
		}
	}
	return false
}

// Save downloads a file referenced by url which is in Card.
func (c *Card) Save(dataRootPath string) error {
	dstPath := path.Join(dataRootPath, c.author)
	err := os.MkdirAll(dstPath, 0777)
	if err != nil {
		return err
	}
	resp, err := http.Get(c.U.String())
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

// RetrieveCards parse HTML referenced by cardPageURLch and create Card instance.
func RetrieveCards(cardPageURLch <-chan *url.URL, cardch chan<- Card, finch chan bool) {
	for cardPageURL := range cardPageURLch {
		resp, err := http.Get(cardPageURL.String())
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
							s := strings.Split(cardPageURL.String(), "/")
							if u, err := url.Parse(strings.Join(s[:len(s)-1], "/") + v.Val[1:]); err != nil {
								log.Fatalln(err)
							} else if u.IsAbs() {
								card.U = u
							}
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
	close(cardch)
	return
}

// RetrieveCardPages retrieves each card page from index.
func RetrieveCardPages(allIndexURLch <-chan *url.URL, cardPageURLch chan<- *url.URL) {
	for allIndexURL := range allIndexURLch {
		resp, err := http.Get(allIndexURL.String())
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
							u, err := url.Parse("http://www.aozora.gr.jp" + strings.Trim(v.Val, ".."))
							if err != nil {
								log.Fatalln(err)
							}
							if u.IsAbs() {
								cardPageURLch <- u
							}
							break
						}
					}
				}
			}
		}
	}
	close(cardPageURLch)
	return
}

// RetrieveAllIndexURLs retrieves all index URLs.
func RetrieveAllIndexURLs(firstIndexURLch <-chan *url.URL, allIndexURLch chan<- *url.URL) {
	for firstIndexURL := range firstIndexURLch {
		resp, err := http.Get(firstIndexURL.String())
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
							u, err := url.Parse("http://www.aozora.gr.jp/index_pages/" + v.Val)
							if err != nil {
								log.Fatalln(err)
							}
							if u.IsAbs() {
								allIndexURLch <- u
							}
							break
						}
					}
				}
			}
		}
	}
	close(allIndexURLch)
	return
}

// RetrieveFirstIndexURLs retrieves first index pages with seed.
func RetrieveFirstIndexURLs(r io.Reader, firstIndexURLch chan *url.URL) {
	insideSakuhinListTable := false
	d := html.NewTokenizer(r)
	for {
		// token type
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			//finch <- true
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
						u, err := url.Parse("http://www.aozora.gr.jp/" + v.Val)
						if err != nil {
							log.Fatalln(err)
						}
						if u.IsAbs() {
							firstIndexURLch <- u
						}
					}
				}
			}
		case html.EndTagToken:
			if token.Data == "table" && insideSakuhinListTable {
				insideSakuhinListTable = false
			}
		}
	}
	close(firstIndexURLch)
	return
}
