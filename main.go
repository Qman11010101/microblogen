package main

import (
	"encoding/json"
	"fmt"
	"html/template"
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
	PageShowLimit int    `json:"pageShowLimit"`
}

type ContentList struct {
	Contents   []Contents `json:"contents"`
	Totalcount int        `json:"totalCount"`
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
}
type Body struct {
	Fieldid string `json:"fieldId"`
	Body    string `json:"body"`
}
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Contents struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Body        []Body     `json:"body"`
	PublishedAt time.Time  `json:"publishedAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	Category    []Category `json:"category"`
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

	articlesJsonPath := Config.Exportpath + "/articles.json"

	client := microcms.New(Config.Servicedomain, Config.Apikey)

	// 先にミニマムな全部入りのやつ落としてcontent数を取得しておく

	// for内に入れる
	var articlesInfo ContentList

	// 公開されていない記事は載らない
	err = client.List(
		microcms.ListParams{
			Endpoint: "article",
			Fields:   []string{"id", "title", "body", "publishedAt", "updatedAt", "category.id", "category.name"},
			Limit:    Config.PageShowLimit,
		}, &articlesInfo)

	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// TODO: あとで消す
	fmt.Println("Latest Contents:")
	for i := 0; i < len(articlesInfo.Contents); i++ {
		fmt.Printf("%+v\n", articlesInfo.Contents[i])
	}
	// TODO: あとで消す

	// 記事のJSONがなければ生成して書き込む
	if !fileExists(articlesJsonPath) {
		articlesFile, err := os.Create(articlesJsonPath)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		defer articlesFile.Close()

		s, err := json.Marshal(articlesInfo.Contents)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		articlesFile.WriteString(string(s))
	}

	var currentArticles []Contents

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

	// とりあえず全レンダリング作ってみる→currentは使わずlatestのみ
	// ヘルパー関数: datetimeのフォーマット
	functionMapping := template.FuncMap{
		"formatTime": func(t time.Time) string { return t.Format("2006-01-02") },
	}
	// トップページ(index.html)レンダリング: articlesInfoを使う
	indexTemplate := template.Must(template.New("index.html").Funcs(functionMapping).ParseFiles(Config.Templatepath + "/index.html"))
	indexOutputFile, err := os.Create(Config.Exportpath + "/index.html")
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	defer indexOutputFile.Close()

	if err = indexTemplate.Execute(indexOutputFile, articlesInfo.Contents); err != nil {
		fmt.Println("Error: ", err)
		return
	}
}
