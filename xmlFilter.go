package diet

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
)

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

func filterXml(uri string, fn func(Item) (bool, error)) (RSS, error) {
	var i RSS
	resp, err := http.Get(uri)
	if err != nil {
		return i, err
	}

	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return i, err
	}

	err = xml.Unmarshal(contents, &i)
	if err != nil {
		return i, err
	}

	var validItems []Item
	var sc = make(chan Item)
	var fc = make(chan bool)

	for _, item := range i.Items.ItemList {
		go check(sc, fc, item, fn)
	}

	for s := 0; s < len(i.Items.ItemList); s++ {
		select {
		case item := <-sc:
			validItems = append(validItems, item)
		case <-fc:
			// pass
		}
	}

	i.Items.ItemList = validItems

	return i, nil
}

func check(sc chan Item, fc chan bool, item Item, fn func(Item) (bool, error)) {
	isGood, err := fn(item)
	if err != nil {
		fc <- false
		return
	}

	if isGood {
		sc <- item
	} else {
		fc <- false
	}
}
