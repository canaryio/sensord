package manifest

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type SiteDefinition struct {
	ID       string   `json:"id"`
	URL      string   `json:"url"`
	Metric   string   `json:"metric"`
	Services []string `json:"services"`
}

type ServiceDefinition struct {
	ID       string            `json:"id"`
	Provider string            `json:"provider"`
	Config   map[string]string `json:"config"`
}

type Manifest struct {
	Sites    []*SiteDefinition    `json:"sites"`
	Services []*ServiceDefinition `json:"services"`
}

// SiteMap returns a map of Site IDs and Sites.
func (m *Manifest) SiteMap() map[string]*SiteDefinition {
	sm := make(map[string]*SiteDefinition)
	for _, site := range m.Sites {
		sm[site.ID] = site
	}
	return sm
}

func Get(url string) (*Manifest, error) {
	manifest := &Manifest{}

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func ContinuouslyGet(url string, ch chan *Manifest) {
	m, err := Get("http://mvp.canary.io/manifests/v2")
	if err != nil {
		log.Print(err)
	}
	ch <- m

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			m, err := Get("http://mvp.canary.io/manifests/v2")
			if err != nil {
				log.Print(err)
			} else {
				ch <- m
			}
		}
	}
}
