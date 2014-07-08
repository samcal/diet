package main

import (
  "github.com/gorilla/mux"
  "net/http"
  "log"
  "io/ioutil"
  "encoding/xml"
  "regexp"
  "strconv"
)

var pointsRegexp = regexp.MustCompile("(\\d+) points")

type RSS struct {
  XMLName xml.Name `xml:"rss"`
  Items Items `xml:"channel"`
}

type Items struct {
  XMLName xml.Name `xml:"channel"`
  ItemList []Item `xml:"item"`
}

type Item struct {
  Title string `xml:"title"`
  Link string `xml:"link"`
  Description string `xml:"description"`
  Comments string `xml:"comments"`
}

func xmlHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/xml")
    fn(w, r)
  }
}

func main() {
  r := mux.NewRouter()
  r.HandleFunc("/", home)
  r.HandleFunc("/feeds/hn/{min_points:[0-9]+}", xmlHandler(hn))

  http.Handle("/", r)

  log.Println("Listening...")
  http.ListenAndServe(":3000", nil)
}

func home(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("Hello!"))
}

func hn(w http.ResponseWriter, r *http.Request) {
  params := mux.Vars(r)
  min_points, _ := strconv.Atoi(params["min_points"])

  resp, err := http.Get("https://news.ycombinator.com/rss")
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadGateway)
    return
  }
  defer resp.Body.Close()
  contents, _ := ioutil.ReadAll(resp.Body)

  var i RSS
  err = xml.Unmarshal(contents, &i)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  var validItems []Item
  for _,item := range i.Items.ItemList {
    resp, err = http.Get(item.Comments)
    if err != nil {
      http.Error(w, err.Error(), http.StatusBadGateway)
      return
    }
    defer resp.Body.Close()
    html, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }
    matches := pointsRegexp.FindStringSubmatch(string(html))
    if len(matches) >= 2 {
      points, err := strconv.Atoi(matches[1])
      if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }
      if points >= min_points {
        validItems = append(validItems, item)
      }
    }
  }

  i.Items.ItemList = validItems

  data, _ := xml.MarshalIndent(i, "", "    ")
  w.Write(data)
}
