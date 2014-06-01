package main

import (
	"log"
	"net/http"
	"net/url"
	"os/user"
	"path"

	"github.com/masatana/aozorago/retriever"
)

func main() {
	downloaded := make(map[*url.URL]bool)
	finch := make(chan bool)
	firstIndexURLch := make(chan *url.URL)
	allIndexURLch := make(chan *url.URL)
	cardPageURLch := make(chan *url.URL)
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	cardch := make(chan retriever.Card)
	resp, err := http.Get("http://www.aozora.gr.jp/")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	go retriever.RetrieveFirstIndexURLs(resp.Body, firstIndexURLch)
	go retriever.RetrieveAllIndexURLs(firstIndexURLch, allIndexURLch)
	go retriever.RetrieveCardPages(allIndexURLch, cardPageURLch)
	go retriever.RetrieveCards(cardPageURLch, cardch, finch)
	for card := range cardch {
		if _, ok := downloaded[card.U]; ok {
			log.Printf("Already downloaded!: %s", card.U)
		} else {
			err := card.Save(path.Join(usr.HomeDir, "/aozora.gr.jp"))
			if err != nil {
				log.Fatalln(err)
			}
			downloaded[card.U] = true
		}

	}
	return
}
