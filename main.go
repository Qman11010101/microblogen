package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/microcmsio/microcms-go-sdk"
	"github.com/otiai10/copy"
)

const VERSION = "2.0.0"

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

func main() {
	var verFlag bool
	flag.BoolVar(&verFlag, "version", false, "show version")
	flag.Parse()

	if verFlag {
		fmt.Println("microblogen v" + VERSION)
		os.Exit(0)
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	log.SetFlags(log.Ltime)
	log.Print("microblogen v" + VERSION)

	cfg, err := LoadConfig()
	if err != nil {
		log.Panic(err)
	}

	// -----------------
	// テンプレート存在確認
	// -----------------
	if !FileExists(cfg.Paths.TemplatesPath) || !FileExists(cfg.Paths.BlogTemplatesPath+"/article.html") || !FileExists(cfg.Paths.BlogTemplatesPath+"/index.html") {
		log.Fatal("Error: Missing templates. You must prepare \"article.html\" and \"index.html\" inside ./" + cfg.Paths.BlogTemplatesPath + ".")
	}

	// ---------------
	// 出力フォルダ再生成
	// ---------------
	if FileExists(cfg.Paths.ExportPath) {
		log.Print(">> Removing existing export directory")
		if err := os.RemoveAll(cfg.Paths.ExportPath); err != nil {
			log.Panic(err)
		}
	}

	// コンポーネント一覧を取得(なければディレクトリだけ生成)
	var componentDirPath = cfg.Paths.TemplatesPath + componentsDirPath
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
	os.MkdirAll(cfg.Paths.ExportPath+"/articles/category/", 0777)

	// TODO
	// staticフォルダ内のコピーを行う
	log.Print(">> Copying static assets to export directory")
	copy.Copy(cfg.Paths.StaticPath, cfg.Paths.ExportPath)

	// microcms用クライアントインスタンス生成
	client := microcms.New(cfg.ServiceDomain, cfg.Apikey)

	// 先にミニマムなlatest用のやつ落としてcontent数(totalCount)を取得できるようにしておく
	var articlesLatest ArticleList

	if err := client.List(
		microcms.ListParams{
			Endpoint: "article",
			Fields:   []string{"id", "title", "publishedAt", "updatedAt", "category.id", "category.name"},
			Limit:    cfg.LatestArticles,
			Orders:   []string{"-publishedAt"},
		}, &articlesLatest,
	); err != nil {
		log.Panic(err)
	}

	// 最新記事のJSONを保存
	saveLatestArticlesJSON(cfg, articlesLatest)

	contentsCount := articlesLatest.Totalcount
	pageLimit := cfg.PageShowLimit
	actualPageCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

	// ヘルパー関数
	helperCtx := HelperContext{Tz: cfg.Tz}
	functionMapping := HelperFunctionsMapping(helperCtx)

	log.Print(">> Parsing templates")
	// テンプレートを事前にパース
	plusIdx := append([]string{cfg.Paths.BlogTemplatesPath + "/index.html"}, componentFilesName...)
	indexTemplate := template.Must(template.New("index.html").Funcs(functionMapping).ParseFiles(plusIdx...))

	plusAtc := append([]string{cfg.Paths.BlogTemplatesPath + "/article.html"}, componentFilesName...)
	articleTemplate := template.Must(template.New("article.html").Funcs(functionMapping).ParseFiles(plusAtc...))

	log.Print(">> Rendering start ")

	// -----------------------------------
	// メインページ(index.html)/記事ページ生成
	// -----------------------------------
	renderMainPagesAndArticles(
		cfg,
		client,
		indexTemplate,
		articleTemplate,
		articlesLatest,
		actualPageCount,
		pageLimit,
		componentFilesName,
		functionMapping,
	)

	// ---------------
	// カテゴリページ生成
	// ---------------
	var categoriesList CategoryList
	if err := client.List(
		microcms.ListParams{
			Endpoint: "category",
			Fields:   []string{"id", "name"},
			Limit:    cfg.MgNum.FreeContentsLimit,
		},
		&categoriesList,
	); err != nil {
		log.Panic(err)
	}

	// カテゴリのJSONを保存
	saveCategoriesJSON(cfg, categoriesList)

	renderCategoryPages(
		cfg,
		client,
		indexTemplate,
		categoriesList,
		pageLimit,
	)

	// ----------------
	// Singlesページ生成
	// ----------------
	renderSinglePages(
		cfg,
		articlesLatest,
		categoriesList,
		componentFilesName,
		functionMapping,
	)

	log.Print("Rendering done!")
}
