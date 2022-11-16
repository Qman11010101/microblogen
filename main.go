package main

import (
	"encoding/json"
	"fmt"
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

	// arguments := os.Args[1:]

	// config.json読み込み
	if !fileExists(configFile) {
		fmt.Println("Error: Missing " + configFile)
		os.Exit(1)
	}

	configFileBytes, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	err = json.Unmarshal([]byte(configFileBytes), &Config)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	client := microcms.New(Config.Servicedomain, Config.Apikey)

	// 新規記事・更新記事のチェック
	var latestContentsInfo ContentList

	// 公開されていない記事は載らない
	err = client.List(
		microcms.ListParams{
			Endpoint: "article",
			// Fields:   []string{"id", "title", "publishedAt", "updatedAt", "category.id", "category.name"},
			Fields: []string{"id", "title", "publishedAt", "updatedAt", "category.id"},
			Limit:  10000, // 無料枠リミット
		}, &latestContentsInfo)

	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// TODO: あとで消す
	fmt.Println("Latest Contents:")
	for i := 0; i < len(latestContentsInfo.Contents); i++ {
		fmt.Printf("%+v\n", latestContentsInfo.Contents[i])
	}
	// TODO: あとで消す

	articlesJsonPath := Config.Exportpath + "/articles.json"

	// 記事のJSONがなければ生成して書き込む
	if !fileExists(articlesJsonPath) {
		articlesFile, err := os.Create(articlesJsonPath)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		defer articlesFile.Close()

		s, err := json.Marshal(latestContentsInfo.Contents)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		articlesFile.WriteString(string(s))
	}

	var currentArticles []Content

	articlesJsonBytes, err := os.ReadFile(articlesJsonPath)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	err = json.Unmarshal([]byte(articlesJsonBytes), &currentArticles)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	fmt.Println("Current Contents:")
	for i := 0; i < len(currentArticles); i++ {
		fmt.Printf("%+v\n", currentArticles[i])
	}

	// 差分レンダリング条件:
	// ・IDに該当する記事がない
	// ・UpdatedAtが違う

	//
}
