package main

import (
	"log"
	"net/http"

	"github.com/masatana/aozorago/retriever"
)

func main() {
	urlch := make(chan string)
	finch := make(chan bool)
	urls := make([]string, 40)
	resp, err := http.Get("http://www.aozora.gr.jp/")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	go retriever.RetrieveFirstIndexUrls(resp.Body, urlch, finch)
FIRSTINDEXLOOP:
	for {
		select {
		case url := <-urlch:
			urls = append(urls, url)
		case <-finch:
			break FIRSTINDEXLOOP
		}
	}
	go retriever.RetrieveAllIndexUrls(urls, urlch, finch)
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
		if !retriever.Contains(allIndexUrls, url) {
			allIndexUrls = append(allIndexUrls, url)
		}
	}
	cardPageUrls := make([]string, 100)
	go retriever.RetrieveCardPages(allIndexUrls, urlch, finch)
ALLCARDPAGES:
	for {
		select {
		case url := <-urlch:
			cardPageUrls = append(cardPageUrls, url)
		case <-finch:
			break ALLCARDPAGES
		}
	}
	cardch := make(chan retriever.Card)
	go retriever.RetrieveCards(cardPageUrls, cardch, finch)
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
