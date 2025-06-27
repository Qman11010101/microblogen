package main

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/microcmsio/microcms-go-sdk"
)

const VERSION = "1.7.0"

const componentsDirPath = "/components"


type ArticleList struct {
	Articles    []Article `json:"contents"`
	Totalcount  int       `json:"totalCount"`
	Offset      int       `json:"offset"`
	Limit       int       `json:"limit"`
	NextPage    int
	PrevPage    int
	AllPage     int
	Root        string
	IsIndex     bool
	ArchiveName string
}
type Body struct {
	Fieldid string `json:"fieldId"`
	Body    string `json:"body"`
}
type Category struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int
}
type Article struct {
	ID          string     `json:"id,omitempty"`
	Title       string     `json:"title,omitempty"`
	Body        []Body     `json:"body,omitempty"`
	PublishedAt time.Time  `json:"publishedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt,omitempty"`
	Category    []Category `json:"category,omitempty"`
	Event       Event      `json:"event,omitempty"`
}
type Event struct {
	EventText string `json:"eventText,omitempty"`
	EventLink string `json:"eventLink,omitempty"`
}
type CategoryList struct {
	Categories []Category `json:"contents,omitempty"`
	Totalcount int        `json:"totalCount,omitempty"`
	Offset     int        `json:"offset,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}



// main section

func main() {
	log.SetFlags(log.Ltime)
	log.Print("microblogen v" + VERSION)

	// ID引数に取って差分レンダリングできそう？
	// arguments := os.Args[1:]

	// ------------------------
	// 設定ファイル/環境変数読み込み
	// ------------------------
	LoadConfig()

	tz, err := time.LoadLocation(Cfg.Timezone)
	if err != nil {
		log.Fatal("Error: Invalid timezone: " + Cfg.Timezone)
	}

	// -----------------
	// テンプレート存在確認
	// -----------------
	if !FileExists(Cfg.Templatepath) || !FileExists(Cfg.Templatepath+"/article.html") || !FileExists(Cfg.Templatepath+"/index.html") {
		log.Fatal("Error: Missing templates. You must prepare \"article.html\" and \"index.html\" inside ./" + Cfg.Templatepath + ".")
	}

	// ---------------
	// 出力フォルダ再生成
	// ---------------
	if FileExists(Cfg.Exportpath) {
		log.Print(">> Removing existing export directory")
		if err := os.RemoveAll(Cfg.Exportpath); err != nil {
			log.Panic(err)
		}
	}

	// コンポーネント一覧を取得(なければディレクトリだけ生成)
	var componentDirPath = Cfg.Templatepath + componentsDirPath
	componentFiles, err := os.ReadDir(componentDirPath)
	if err != nil {
		log.Print("Warning: Components directory not found. The directory will be automatically generated.")
		if err := os.Mkdir(componentDirPath, 0755); err != nil {
			log.Panic(err)
		}
	}

	var componentFilesName []string

	for _, file := range componentFiles {
		if !file.IsDir() {
			componentFilesName = append(componentFilesName, componentDirPath+"/"+file.Name())
		}
	}

	log.Print(">> Generating export directory")
	os.MkdirAll(Cfg.Exportpath+"/articles/category/", 0777)

	// TODO
	// .mbignoreに記載されたファイル：除外
	// 記載されていないhtmlファイル：レンダリング（レンダリングされる部分がない場合はそのファイルがコピーされる挙動にする）
	// 記載されていない非htmlファイル：コピー（画像やCSSなど）
	// config.jsonにテンプレートの拡張子を指定できるようにしてもいいかも　デフォルトはhtml
	// componentsディレクトリも除外対象ではあるので、ここらへんをもう少しわかりやすい形にしたい

	// microcms用クライアントインスタンス生成
	client := microcms.New(Cfg.Servicedomain, Cfg.Apikey)

	// 先にミニマムなlatest用のやつ落としてcontent数(totalCount)を取得できるようにしておく
	var articlesLatest ArticleList

	if err := client.List(
		microcms.ListParams{
			Endpoint: "article",
			Fields:   []string{"id", "title", "publishedAt", "updatedAt", "category.id", "category.name"},
			Limit:    DEFAULT_LATEST_ARTICLES,
			Orders:   []string{"-publishedAt"},
		}, &articlesLatest,
	); err != nil {
		log.Panic(err)
	}

	// 最新記事のJSONを保存
	articlesFile, err := os.Create(Cfg.Exportpath + "/latest.json")
	if err != nil {
		log.Panic(err)
	}
	defer articlesFile.Close()

	s, err := json.Marshal(articlesLatest.Articles)
	if err != nil {
		log.Panic(err)
	}
	articlesFile.WriteString(string(s))

	contentsCount := articlesLatest.Totalcount
	pageLimit := Cfg.PageShowLimit
	loopsCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

	// -----------------------------------
	// メインページ(index.html)/記事ページ生成
	// -----------------------------------

	// ヘルパー関数
	functionMapping := HelperFunctionsMapping(tz)

	log.Print(">> Rendering start ")

	var mu sync.Mutex
	articleCounter := 0

	for i := 0; i < loopsCount; i++ {
		log.Print("Rendering mainpage ", i+1, " / ", loopsCount)
		var articlesPart ArticleList

		if err := client.List(
			microcms.ListParams{
				Endpoint: "article",
				Fields:   []string{"id", "title", "event", "body", "publishedAt", "updatedAt", "category.id", "category.name"},
				Limit:    pageLimit,
				Offset:   pageLimit * i,
				Orders:   []string{"-publishedAt"},
			}, &articlesPart,
		); err != nil {
			log.Panic(err)
		}

		articlesPart.NextPage = i + 2
		articlesPart.PrevPage = i
		articlesPart.AllPage = loopsCount
		articlesPart.Root = "/"
		articlesPart.IsIndex = true

		// トップページ(index.html)レンダリング
		plusIdx := append([]string{Cfg.Templatepath + "/index.html"}, componentFilesName...)
		indexTemplate := template.Must(template.New("index.html").Funcs(functionMapping).ParseFiles(plusIdx...))
		var outputFilePath string
		if i == 0 {
			outputFilePath = Cfg.Exportpath + "/index.html"
		} else {
			outputBasePath := Cfg.Exportpath + "/page/" + strconv.Itoa(i+1)
			os.MkdirAll(outputBasePath, 0755)
			outputFilePath = outputBasePath + "/index.html"
		}
		indexOutputFile, err := os.Create(outputFilePath)
		if err != nil {
			log.Panic(err)
		}
		defer indexOutputFile.Close()

		if err := indexTemplate.Execute(indexOutputFile, articlesPart); err != nil {
			log.Panic(err)
		}

		// 記事レンダリング
		var wgArticles sync.WaitGroup

		for a := 0; a < len(articlesPart.Articles); a++ {
			wgArticles.Add(1)
			go func(i int, a int) {
				plusAtc := append([]string{Cfg.Templatepath + "/article.html"}, componentFilesName...)
				articleTemplate := template.Must(template.New("article.html").Funcs(functionMapping).ParseFiles(plusAtc...))
				outputFilePath := Cfg.Exportpath + "/articles/" + articlesPart.Articles[a].ID + ".html"
				articleOutputFile, err := os.Create(outputFilePath)
				if err != nil {
					log.Panic(err)
				}
				defer articleOutputFile.Close()

				if err := articleTemplate.Execute(articleOutputFile, articlesPart.Articles[a]); err != nil {
					log.Panic(err)
				}

				mu.Lock()
				articleCounter++
				log.Print("- Rendered articles ", articleCounter, " / ", articlesPart.Totalcount)
				mu.Unlock()
				wgArticles.Done()
			}(i, a)
		}
		wgArticles.Wait()
	}

	// ---------------
	// カテゴリページ生成
	// ---------------

	// カテゴリ(タグ)の構造体
	var categoriesList CategoryList

	if err := client.List(
		microcms.ListParams{
			Endpoint: "category",
			Fields:   []string{"id", "name"},
			Limit:    FREE_CONTENTS_LIMIT,
		},
		&categoriesList,
	); err != nil {
		log.Panic(err)
	}

	// カテゴリのJSONを保存
	categoriesFile, err := os.Create(Cfg.Exportpath + "/category.json")
	if err != nil {
		log.Panic(err)
	}
	defer categoriesFile.Close()

	x, err := json.Marshal(categoriesList.Categories)
	if err != nil {
		log.Panic(err)
	}
	categoriesFile.WriteString(string(x))

	// カテゴリレンダリング
	categories := categoriesList.Categories
	var wgCategories sync.WaitGroup
	categoryCounter := 0
	for c := 0; c < len(categories); c++ {
		wgCategories.Add(1)
		go func(c int) {
			var categoryArticlesMinimum ArticleList
			categoryID := categories[c].ID

			if err := client.List(
				microcms.ListParams{
					Endpoint: "article",
					Fields:   []string{"id"},
					Limit:    FREE_CONTENTS_LIMIT,
					Orders:   []string{"-publishedAt"},
					Filters:  "category[contains]" + categoryID,
				}, &categoryArticlesMinimum,
			); err != nil {
				log.Panic(err)
			}

			contentsCount := categoryArticlesMinimum.Totalcount
			loopsCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

			categoryOutputBasePath := Cfg.Exportpath + "/articles/category/" + categoryID
			os.MkdirAll(categoryOutputBasePath, 0755)

			for i := 0; i < loopsCount; i++ {
				var categoryArticlesPart ArticleList

				if err := client.List(
					microcms.ListParams{
						Endpoint: "article",
						Fields:   []string{"id", "title", "body", "event", "publishedAt", "updatedAt", "category.id", "category.name"},
						Limit:    pageLimit,
						Offset:   pageLimit * i,
						Filters:  "category[contains]" + categoryID,
					}, &categoryArticlesPart,
				); err != nil {
					log.Panic(err)
				}

				categoryArticlesPart.NextPage = i + 2
				categoryArticlesPart.PrevPage = i
				categoryArticlesPart.AllPage = loopsCount
				categoryArticlesPart.Root = "/articles/category/" + categoryID + "/"
				categoryArticlesPart.IsIndex = false
				categoryArticlesPart.ArchiveName = Cfg.CategoryTagName + ": " + categories[c].Name

				// カテゴリのトップページ(index.html)レンダリング
				plusCatIdx := append([]string{Cfg.Templatepath + "/index.html"}, componentFilesName...)
				categoryIndexTemplate := template.Must(template.New("index.html").Funcs(functionMapping).ParseFiles(plusCatIdx...))
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
					log.Panic(err)
				}
				defer indexOutputFile.Close()

				if err := categoryIndexTemplate.Execute(indexOutputFile, categoryArticlesPart); err != nil {
					log.Panic(err)
				}
			}

			mu.Lock()
			categoryCounter++
			mu.Unlock()
			log.Print("Rendered category ", categoryCounter, " / ", len(categories), " '"+categoryID+"'")
			wgCategories.Done()
		}(c)
	}

	wgCategories.Wait()

	log.Print("Rendering done!")
}
