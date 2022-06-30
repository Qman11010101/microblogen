package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/microcmsio/microcms-go-sdk"
)

const configFile = "config.json"
const VERSION = "0.0.1"

type ConfigJson struct {
	Apikey        string `json:"APIkey"`
	Servicedomain string `json:"serviceDomain"`
	Exportpath    string `json:"exportPath"`
	Templatepath  string `json:"templatePath"`
}

type Content struct {
	ID          string    `json:"id,omitempty"`
	Title       string    `json:"title,omitempty"`
	Body        string    `json:"body,omitempty"`
	PublishedAt time.Time `json:"publishedAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

type ContentList struct {
	Contents   []Content
	TotalCount int
	Limit      int
	Offset     int
}

var Config ConfigJson

// Utility Function

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

// main section

func main() {
	fmt.Println("microblogen v" + VERSION)

	// Loading config.json
	if !fileExists(configFile) {
		fmt.Println("Error: Missing " + configFile)
		os.Exit(1)
	}

	f, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	err = json.Unmarshal([]byte(f), &Config)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	client := microcms.New(Config.Servicedomain, Config.Apikey)

	// 新規記事・更新記事のチェック
	var minimumContent ContentList

	err = client.List(
		microcms.ListParams{
			Endpoint: "article",
			// Fields:   []string{"id", "title", "publishedAt", "updatedAt", "category.id", "category.name"},
			Fields: []string{"id", "updatedAt"},
			Limit:  10000, // 無料枠リミット
		}, &minimumContent)

	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	fmt.Printf("%+v\n", minimumContent)

	// 古いJSONがなければこれのcontentsだけ保存して、あればそれと比較して更新があればレンダリングする
}
