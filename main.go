package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"time"

	"github.com/microcmsio/microcms-go-sdk"
)

const configFile = "config.json"
const VERSION = "0.0.1"

type ConfigStruct struct {
	Apikey        string `json:"APIkey"`
	Servicedomain string `json:"serviceDomain"`
	Exportpath    string `json:"exportPath"`
	Templatepath  string `json:"templatePath"`
	PageShowLimit int    `json:"pageShowLimit"`
}

type ContentList struct {
	Contents   []Content `json:"contents"`
	Totalcount int       `json:"totalCount"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
}
type Body struct {
	Fieldid string `json:"fieldId"`
	Body    string `json:"body"`
}
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Content struct {
	ID          string     `json:"id,omitempty"`
	Title       string     `json:"title,omitempty"`
	Body        []Body     `json:"body,omitempty"`
	PublishedAt time.Time  `json:"publishedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt,omitempty"`
	Category    []Category `json:"category,omitempty"`
}

var Config ConfigStruct

// Utility Function

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

// main section

func main() {
	fmt.Println("microblogen v" + VERSION)

	// arguments := os.Args[1:]

	// config.json読み込み→なかったら環境変数から読み込む
	if fileExists(configFile) {
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

	} else {
		fmt.Println("Error: Missing " + configFile)
		os.Exit(1)
	}

	articlesJsonPath := Config.Exportpath + "/articles.json"

	client := microcms.New(Config.Servicedomain, Config.Apikey)

	// 先にミニマムなlatest用のやつ落としてcontent数を取得しておく
	var articlesLatest ContentList

	err := client.List(
		microcms.ListParams{
			Endpoint: "article",
			Fields:   []string{"id", "title", "publishedAt", "updatedAt", "category.id", "category.name"},
			Limit:    5,
			Orders:   []string{"-publishedAt"},
		}, &articlesLatest)

	if err != nil {
		panic(err)
	}

	contentsCount := articlesLatest.Totalcount
	loopsCount := int(math.Ceil(float64(contentsCount) / float64(Config.PageShowLimit)))

	for i := 0; i < loopsCount; i++ {

	}

	// for内に入れる
	var articlesInfo ContentList

	err = client.List(
		microcms.ListParams{
			Endpoint: "article",
			Fields:   []string{"id", "title", "body", "publishedAt", "updatedAt", "category.id", "category.name"},
			Limit:    Config.PageShowLimit,
		}, &articlesInfo)

	if err != nil {
		panic(err)
	}

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
			panic(err)
		}
		articlesFile.WriteString(string(s))
	}

	var currentArticles []Content

	articlesJsonBytes, err := os.ReadFile(articlesJsonPath)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal([]byte(articlesJsonBytes), &currentArticles)
	if err != nil {
		panic(err)
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
		panic(err)
	}
	defer indexOutputFile.Close()

	if err = indexTemplate.Execute(indexOutputFile, articlesInfo.Contents); err != nil {
		panic(err)
	}
}
