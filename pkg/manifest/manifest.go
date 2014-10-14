package manifest

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func Get(url string) (map[string]*string, error) {
	manifest := make(map[string]*string)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func ContinuouslyGet(url string, ch chan map[string]*string) {
	m, err := Get(url)
	if err != nil {
		log.Print(err)
	}
	ch <- m

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			m, err := Get(url)
			if err != nil {
				log.Print(err)
			} else {
				ch <- m
			}
		}
	}
}
