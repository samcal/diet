package main

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"

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

func home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello!"))
}

func checkPoints(sc chan Item, fc chan bool, item Item, minPoints int) {
	response, error := http.Get(item.Comments)
	if error != nil {
		// http.Error(w, error.Error(), http.StatusBadGateway)
		fc <- false
		return
	}
	defer response.Body.Close()

	html, error := ioutil.ReadAll(response.Body)
	if error != nil {
		// http.Error(w, error.Error(), http.StatusInternalServerError)
		fc <- false
		return
	}

	matches := pointsRegexp.FindStringSubmatch(string(html))

	if len(matches) >= 2 {
		points, error := strconv.Atoi(matches[1])
		if error != nil {
			// 	http.Error(w, error.Error(), http.StatusInternalServerError)
			fc <- false
			return
		}

		if points >= minPoints {
			sc <- item
		} else {
			fc <- false
		}
	} else {
		fc <- false
	}
}

func hn(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	min_points, err := strconv.Atoi(params["min_points"])
	if err != nil {
		http.Error(w, "Provided min_points must be an integer", http.StatusBadRequest)
		return
	}

	resp, err := http.Get("https://news.ycombinator.com/rss")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading YC response", http.StatusInternalServerError)
		return
	}

	var i RSS
	err = xml.Unmarshal(contents, &i)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var validItems []Item
	var sc = make(chan Item)
	var fc = make(chan bool)

	for _, item := range i.Items.ItemList {
		go checkPoints(sc, fc, item, min_points)
	}

	var s int
	for s < len(i.Items.ItemList) {

		select {
		case item := <-sc:
			validItems = append(validItems, item)
		case <-fc:
			// pass
		}

		s++
	}

	i.Items.ItemList = validItems

	data, err := xml.MarshalIndent(i, "", "    ")
	if err != nil {
		http.Error(w, "Error expanding to XML: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
