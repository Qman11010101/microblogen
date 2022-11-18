package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/microcmsio/microcms-go-sdk"
	"github.com/otiai10/copy"
)

const configFile = "config.json"
const VERSION = "0.0.1"

type ConfigStruct struct {
	Apikey        string `json:"APIkey"`
	Servicedomain string `json:"serviceDomain"`
	Exportpath    string `json:"exportPath"`
	Templatepath  string `json:"templatePath"`
	AssetsDirName string `json:"assetsDirName"`
	PageShowLimit int    `json:"pageShowLimit"`
}

type ContentList struct {
	Contents   []Content `json:"contents"`
	Totalcount int       `json:"totalCount"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
	NextPage   int
	PrevPage   int
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
			panic(err)
		}

		err = json.Unmarshal([]byte(configFileBytes), &Config)
		if err != nil {
			panic(err)
		}

	} else {
		fmt.Println("Error: Missing " + configFile)
		os.Exit(1)
	}

	if !fileExists(Config.Templatepath) || !fileExists(Config.Templatepath+"/article.html") || !fileExists(Config.Templatepath+"/index.html") {
		fmt.Println("Error: Missing templates. You must prepare \"article.html\" and \"index.html\" inside ./" + Config.Templatepath + ".")
		os.Exit(1)
	}

	// 出力フォルダ削除
	if fileExists(Config.Exportpath) {
		if err := os.RemoveAll(Config.Exportpath); err != nil {
			panic(err)
		}
	}

	// 出力フォルダ生成
	os.Mkdir(Config.Exportpath, 0777)
	os.Mkdir(Config.Exportpath+"/articles", 0777)

	// アセットのコピー
	if fileExists(Config.Templatepath + "/" + Config.AssetsDirName) {
		err := copy.Copy(Config.Templatepath+"/"+Config.AssetsDirName, Config.Exportpath+"/"+Config.AssetsDirName)
		if err != nil {
			panic(err)
		}
	}

	latestArticlesJsonPath := Config.Exportpath + "/latest.json"

	client := microcms.New(Config.Servicedomain, Config.Apikey)

	// 先にミニマムなlatest用のやつ落としてcontent数(totalCount)を取得できるようにしておく
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

	// 最新記事のJSONを入れておく
	articlesFile, err := os.Create(latestArticlesJsonPath)
	if err != nil {
		panic(err)
	}
	defer articlesFile.Close()

	s, err := json.Marshal(articlesLatest.Contents)
	if err != nil {
		panic(err)
	}
	articlesFile.WriteString(string(s))

	// TODO: 先にタグ記事/時期記事生成しとく？

	contentsCount := articlesLatest.Totalcount
	pageLimit := Config.PageShowLimit
	loopsCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

	for i := 0; i < loopsCount; i++ {
		var articlesPart ContentList

		err := client.List(
			microcms.ListParams{
				Endpoint: "article",
				Fields:   []string{"id", "title", "body", "publishedAt", "updatedAt", "category.id", "category.name"},
				Limit:    pageLimit,
				Offset:   pageLimit * i,
				Orders:   []string{"-publishedAt"},
			}, &articlesPart)

		if err != nil {
			panic(err)
		}

		articlesPart.NextPage = i + 2
		articlesPart.PrevPage = i

		// ヘルパー関数
		functionMapping := template.FuncMap{
			"formatTime":   func(t time.Time) string { return t.Format("2006-01-02") },
			"totalGreater": func(total, limit int) bool { return total > limit },
			"isNotFirst":   func(offset int) bool { return offset != 0 },
			"isNotLast":    func(limit, offset, total int) bool { return limit+offset < total },
		}

		// トップページ(index.html)レンダリング
		indexTemplate := template.Must(template.New("index.html").Funcs(functionMapping).ParseFiles(Config.Templatepath + "/index.html"))
		var outputFilePath string
		if i == 0 {
			outputFilePath = Config.Exportpath + "/index.html"
		} else {
			outputBasePath := Config.Exportpath + "/page/" + strconv.Itoa(i+1)
			os.MkdirAll(outputBasePath, 0755)
			outputFilePath = outputBasePath + "/index.html"
		}
		indexOutputFile, err := os.Create(outputFilePath)
		if err != nil {
			panic(err)
		}
		defer indexOutputFile.Close()

		if err := indexTemplate.Execute(indexOutputFile, articlesPart); err != nil {
			panic(err)
		}

		// 記事(article.html)レンダリング
		for a := 0; a < len(articlesPart.Contents); a++ {
			articleTemplate := template.Must(template.New("article.html").Funcs(functionMapping).ParseFiles(Config.Templatepath + "/article.html"))
			outputFilePath := Config.Exportpath + "/articles/" + articlesPart.Contents[a].ID + ".html"
			articleOutputFile, err := os.Create(outputFilePath)
			if err != nil {
				panic(err)
			}
			defer articleOutputFile.Close()

			if err := articleTemplate.Execute(articleOutputFile, articlesPart.Contents[a]); err != nil {
				panic(err)
			}

		}
	}
}
