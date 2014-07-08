package main

import (
	"encoding/xml"
	"log"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
)

var pointsRegexp = regexp.MustCompile("(\\d+) points")

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Items   Items    `xml:"channel"`
}

type Items struct {
	XMLName  xml.Name `xml:"channel"`
	ItemList []Item   `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Comments    string `xml:"comments"`
}

func xmlHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		fn(w, r)
	}
}

func main() {
	root := mux.NewRouter()
	root.HandleFunc("/", home)
	http.Handle("/", root)

	feeds := root.PathPrefix("/feeds").Subrouter()
	feeds.HandleFunc("/hn/{min_points:[0-9]+}", xmlHandler(hn))

	log.Println("Listening...")
	http.ListenAndServe(":3000", nil)
}
