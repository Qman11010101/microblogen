package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"text/template"
	"time"

	"github.com/microcmsio/microcms-go-sdk"
	"github.com/otiai10/copy"
)

const configFile = "config.json"
const VERSION = "0.1"

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
	AllPage    int
	Root       string
}
type Body struct {
	Fieldid string `json:"fieldId"`
	Body    string `json:"body"`
}
type Category struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Articles []Content
}
type Content struct {
	ID          string     `json:"id,omitempty"`
	Title       string     `json:"title,omitempty"`
	Body        []Body     `json:"body,omitempty"`
	PublishedAt time.Time  `json:"publishedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt,omitempty"`
	Category    []Category `json:"category,omitempty"`
}

type ContentCategoryList struct {
	Contents   []Category `json:"contents,omitempty"`
	Totalcount int        `json:"totalCount,omitempty"`
	Offset     int        `json:"offset,omitempty"`
	Limit      int        `json:"limit,omitempty"`
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

	// ID引数に取って差分レンダリングできそう？
	// arguments := os.Args[1:]

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
		fmt.Println(configFile + " not found. Loading the setting values from environment variables...")
		Apikey, ok := os.LookupEnv("Apikey")
		if ok {
			Config.Apikey = Apikey
		} else {
			fmt.Println("Error: Environment variable 'MICROCMS_APIKEY' not found.")
			os.Exit(1)
		}
		Servicedomain, ok := os.LookupEnv("Servicedomain")
		if ok {
			Config.Servicedomain = Servicedomain
		} else {
			fmt.Println("Error: Environment variable 'Servicedomain' not found.")
			os.Exit(1)
		}
		Exportpath, ok := os.LookupEnv("Exportpath")
		if ok {
			Config.Exportpath = Exportpath
		} else {
			Config.Exportpath = "./output"
		}
		Templatepath, ok := os.LookupEnv("Templatepath")
		if ok {
			Config.Templatepath = Templatepath
		} else {
			Config.Templatepath = "./template"
		}
		AssetsDirName, ok := os.LookupEnv("AssetsDirName")
		if ok {
			Config.AssetsDirName = AssetsDirName
		} else {
			Config.AssetsDirName = "assets"
		}
		PageShowLimit, ok := os.LookupEnv("PageShowLimit")
		if ok {
			value, err := strconv.Atoi(PageShowLimit)
			if err != nil {
				fmt.Println("Warning: Environment variable 'PageShowLimit' is not integer; Use default value.")
				Config.PageShowLimit = 10
			} else {
				Config.PageShowLimit = value
			}
		} else {
			Config.PageShowLimit = 10
		}
	}

	if !fileExists(Config.Templatepath) || !fileExists(Config.Templatepath+"/article.html") || !fileExists(Config.Templatepath+"/index.html") {
		fmt.Println("Error: Missing templates. You must prepare \"article.html\" and \"index.html\" inside ./" + Config.Templatepath + ".")
		os.Exit(1)
	}

	// 出力フォルダ削除
	if fileExists(Config.Exportpath) {
		fmt.Println("Removing former export directory...")
		if err := os.RemoveAll(Config.Exportpath); err != nil {
			panic(err)
		}
	}

	// 出力フォルダ生成
	fmt.Println("Generating export directory...")
	os.MkdirAll(Config.Exportpath+"/articles/category/", 0777)

	// アセットのコピー
	if fileExists(Config.Templatepath + "/" + Config.AssetsDirName) {
		fmt.Println("Copying assets...")
		err := copy.Copy(Config.Templatepath+"/"+Config.AssetsDirName, Config.Exportpath+"/"+Config.AssetsDirName)
		if err != nil {
			panic(err)
		}
	}

	latestArticlesJsonPath := Config.Exportpath + "/latest.json"
	categoriesJsonPath := Config.Exportpath + "/category.json"

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

	// 最新記事のJSONを保存
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

	// カテゴリ(タグ)の構造体
	var categoriesList ContentCategoryList

	err = client.List(
		microcms.ListParams{
			Endpoint: "category",
			Fields:   []string{"id", "name"},
			Limit:    10000, // 無料枠リミット
		},
		&categoriesList,
	)
	if err != nil {
		panic(err)
	}

	// カテゴリ(タグ)のJSONを保存
	categoriesFile, err := os.Create(categoriesJsonPath)
	if err != nil {
		panic(err)
	}
	defer categoriesFile.Close()

	x, err := json.Marshal(categoriesList.Contents)
	if err != nil {
		panic(err)
	}
	categoriesFile.WriteString(string(x))

	contentsCount := articlesLatest.Totalcount
	pageLimit := Config.PageShowLimit
	loopsCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

	// trim用正規表現
	htmlTagTrimReg := regexp.MustCompile(`<.*?>`)

	// ヘルパー関数
	functionMapping := template.FuncMap{
		"formatTime":   func(t time.Time) string { return t.Format("2006-01-02") },
		"totalGreater": func(total, limit int) bool { return total > limit },
		"isNotFirst":   func(offset int) bool { return offset != 0 },
		"isNotLast":    func(limit, offset, total int) bool { return limit+offset < total },
		"trimSample": func(body string) string {
			r := []rune(htmlTagTrimReg.ReplaceAllString(body, ""))
			return string(r[:int(math.Min(100, float64(len(r))))]) + "…"
		},
		"sub": func(a, b int) int { return a - b },
	}

	fmt.Println("Rendering...")
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
		articlesPart.AllPage = loopsCount
		articlesPart.Root = "/"

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

	// タグAPI叩いて配列回して生成していく
	// カテゴリレンダリング
	for i := 0; i < len(categoriesList.Contents); i++ {
		var categoryArticlesMinimum ContentList
		categoryID := categoriesList.Contents[i].ID

		err := client.List(
			microcms.ListParams{
				Endpoint: "article",
				Fields:   []string{"id"},
				Limit:    10000, // 無料枠リミット
				Orders:   []string{"-publishedAt"},
				Filters:  "category[contains]" + categoryID,
			}, &categoryArticlesMinimum)

		if err != nil {
			panic(err)
		}

		contentsCount := categoryArticlesMinimum.Totalcount
		loopsCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

		categoryOutputBasePath := Config.Exportpath + "/articles/category/" + categoryID
		os.MkdirAll(categoryOutputBasePath, 0755)

		for i := 0; i < loopsCount; i++ {
			var categoryArticlesPart ContentList

			err := client.List(
				microcms.ListParams{
					Endpoint: "article",
					Fields:   []string{"id", "title", "body", "publishedAt", "updatedAt", "category.id", "category.name"},
					Limit:    pageLimit,
					Offset:   pageLimit * i,
					Filters:  "category[contains]" + categoryID,
				}, &categoryArticlesPart)

			if err != nil {
				panic(err)
			}

			categoryArticlesPart.NextPage = i + 2
			categoryArticlesPart.PrevPage = i
			categoryArticlesPart.AllPage = loopsCount
			categoryArticlesPart.Root = "/articles/category/" + categoryID + "/"

			// カテゴリのトップページ(index.html)レンダリング
			categoryIndexTemplate := template.Must(template.New("index.html").Funcs(functionMapping).ParseFiles(Config.Templatepath + "/index.html"))
			var categoryOutputFilePath string
			if i == 0 {
				categoryOutputFilePath = categoryOutputBasePath + "/index.html"
			} else {
				basePath := categoryOutputBasePath + "/page/" + strconv.Itoa(i+1)
				os.MkdirAll(basePath, 0755)
				categoryOutputFilePath = basePath + "/index.html"
			}
			indexOutputFile, err := os.Create(categoryOutputFilePath)
			if err != nil {
				panic(err)
			}
			defer indexOutputFile.Close()

			if err := categoryIndexTemplate.Execute(indexOutputFile, categoryArticlesPart); err != nil {
				panic(err)
			}
		}
	}

	fmt.Println("Rendering done!")
}
