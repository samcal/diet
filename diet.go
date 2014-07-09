package diet

import (
	"errors"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
)

var pointsRegexp = regexp.MustCompile("(\\d+) points")

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

func hn(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	min_points, err := strconv.Atoi(params["min_points"])
	if err != nil {
		http.Error(w, "Provided min_points must be an integer", http.StatusBadRequest)
		return
	}

	i, err := filterXml("https://news.ycombinator.com/rss", func(item Item) (bool, error) {
		resp, err := http.Get(item.Comments)
		if err != nil {
			return false, err
		}

		defer resp.Body.Close()
		html, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}

		matches := pointsRegexp.FindStringSubmatch(string(html))
		if len(matches) >= 2 {
			points, err := strconv.Atoi(matches[1])
			if err != nil {
				return false, err
			}

			if points >= min_points {
				return true, nil
			} else {
				return false, nil
			}
		} else {
			return false, errors.New("No text matched the points regex")
		}
	})

	data, err := xml.MarshalIndent(i, "", "    ")
	if err != nil {
		http.Error(w, "Error expanding to XML: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
