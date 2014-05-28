package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"code.google.com/p/go.net/html"
)

func parseHTML(r io.Reader, urlch chan string, finch chan bool) {
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
						urlch <- v.Val
					}
				}
			}
		case html.EndTagToken:
			if token.Data == "table" && insideSakuhinListTable {
				insideSakuhinListTable = false
			}
		}
	}
}

func main() {
	urlch := make(chan string)
	finch := make(chan bool)
	resp, err := http.Get("http://www.aozora.gr.jp/")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	go parseHTML(resp.Body, urlch, finch)
LOOP:
	for {
		select {
		case url := <-urlch:
			fmt.Println(url)
		case <-finch:
			break LOOP
		}
	}
	return
	//body, err := ioutil.ReadAll(resp.Body)
	//fmt.Printf("%T", body)
}
