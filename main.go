package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"text/template"
	"time"

	"io/fs"

	"github.com/joho/godotenv"
	"github.com/microcmsio/microcms-go-sdk"
	"github.com/otiai10/copy"
)

const VERSION = "2.0.1"

const (
	INDEX_HTML   = "index.html"
	ARTICLE_HTML = "article.html"
)

type ArticleList struct {
	Articles    []Article `json:"contents"`
	Totalcount  int       `json:"totalCount"`
	Offset      int       `json:"offset"`
	Limit       int       `json:"limit"`
	NextPage    int
	CurrentPage int
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
	ID   string `json:"id"`
	Name string `json:"name"`
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

	_ = godotenv.Load()

	log.SetFlags(log.Ltime)
	log.Print("microblogen v" + VERSION)

	log.Print(">> Loading the setting values from environment variables")
	cfg, err := LoadConfig()
	if err != nil {
		log.Panic(err)
	}

	if !PathExists(cfg.Paths.ResourcesPath) {
		// リソースディレクトリが存在しない場合は作成して終了
		log.Print("Resources directory not found. Creating resources directory at " + cfg.Paths.ResourcesPath)
		if err := os.MkdirAll(cfg.Paths.StaticPath, 0755); err != nil {
			log.Panic(err)
		}
		if err := os.MkdirAll(cfg.Paths.BlogTemplatesPath, 0755); err != nil {
			log.Panic(err)
		}
		if err := os.MkdirAll(cfg.Paths.SinglesTemplatesPath, 0755); err != nil {
			log.Panic(err)
		}
		if err := os.MkdirAll(cfg.Paths.ComponentsPath, 0755); err != nil {
			log.Panic(err)
		}
		log.Print("Resources directory created successfully.")
		log.Print("Please prepare the templates and static files in the resources directory.")
		os.Exit(0)
	}

	// ---------------
	// 出力フォルダ再生成
	// ---------------
	if PathExists(cfg.Paths.ExportPath) {
		log.Print(">> Removing existing export directory")
		if err := os.RemoveAll(cfg.Paths.ExportPath); err != nil {
			log.Panic(err)
		}
	}

	// コンポーネント一覧を取得(なければディレクトリだけ生成)
	var componentDirPath = cfg.Paths.ComponentsPath
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
	articlesFile, err := os.Create(cfg.Paths.ExportPath + "/latest.json")
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
	pageLimit := cfg.ArticlesPerPage
	actualPageCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

	// -----------------------------------
	// メインページ(index.html)/記事ページ生成
	// -----------------------------------

	// ヘルパー関数
	helperCtx := HelperContext{Tz: cfg.Tz}
	functionMapping := HelperFunctionsMapping(helperCtx)

	log.Print(">> Parsing templates")

	plusIdx := append([]string{cfg.Paths.BlogTemplatesPath + "/" + INDEX_HTML}, componentFilesName...)
	indexTemplate := template.Must(template.New(INDEX_HTML).Funcs(functionMapping).ParseFiles(plusIdx...))

	plusAtc := append([]string{cfg.Paths.BlogTemplatesPath + "/" + ARTICLE_HTML}, componentFilesName...)
	articleTemplate := template.Must(template.New(ARTICLE_HTML).Funcs(functionMapping).ParseFiles(plusAtc...))

	log.Print(">> Rendering start ")

	var mu sync.Mutex
	articleCounter := 0

	for i := 0; i < actualPageCount; i++ {
		log.Print("Rendering mainpage ", i+1, " / ", actualPageCount)
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
		articlesPart.CurrentPage = i + 1
		articlesPart.PrevPage = i
		articlesPart.AllPage = actualPageCount
		articlesPart.Root = "/"
		articlesPart.IsIndex = true

		// トップページ(index.html)レンダリング
		var outputFilePath string
		if i == 0 {
			outputFilePath = cfg.Paths.ExportPath + "/" + INDEX_HTML
		} else {
			outputBasePath := cfg.Paths.ExportPath + "/page/" + strconv.Itoa(i+1)
			os.MkdirAll(outputBasePath, 0755)
			outputFilePath = outputBasePath + "/" + INDEX_HTML
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
				outputFilePath := cfg.Paths.ExportPath + "/articles/" + articlesPart.Articles[a].ID + ".html"
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
			Limit:    10000,
		},
		&categoriesList,
	); err != nil {
		log.Panic(err)
	}

	// カテゴリレンダリング
	categories := categoriesList.Categories
	var wgCategories sync.WaitGroup
	categoryCounter := 0

	// 削除対象カテゴリIDを格納するスライスとミューテックス
	var noArticleCategoryIDs []string
	var noArticleCategoryIDsMu sync.Mutex

	for c := 0; c < len(categories); c++ {
		wgCategories.Add(1)
		go func(c int) {
			var categoryArticlesMinimum ArticleList
			categoryID := categories[c].ID
			categoryName := categories[c].Name

			if err := client.List(
				microcms.ListParams{
					Endpoint: "article",
					Fields:   []string{"id"},
					Limit:    10000,
					Orders:   []string{"-publishedAt"},
					Filters:  "category[contains]" + categoryID,
				}, &categoryArticlesMinimum,
			); err != nil {
				log.Panic(err)
			}

			contentsCount := categoryArticlesMinimum.Totalcount
			if contentsCount == 0 {
				noArticleCategoryIDsMu.Lock()
				noArticleCategoryIDs = append(noArticleCategoryIDs, categoryID)
				noArticleCategoryIDsMu.Unlock()
				mu.Lock()
				categoryCounter++
				mu.Unlock()
				log.Print("No articles found in category ", categoryCounter, " / ", len(categories), " '", categoryID, "'. Skipped rendering.")
				wgCategories.Done()
				return
			}
			loopsCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

			categoryOutputBasePath := cfg.Paths.ExportPath + "/articles/category/" + categoryID
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
				categoryArticlesPart.CurrentPage = i + 1
				categoryArticlesPart.PrevPage = i
				categoryArticlesPart.AllPage = loopsCount
				categoryArticlesPart.Root = "/articles/category/" + categoryID + "/"
				categoryArticlesPart.IsIndex = false
				categoryArticlesPart.ArchiveName = cfg.CategoryTagName + ": " + categoryName

				// カテゴリのトップページ(index.html)レンダリング
				var categoryOutputFilePath string
				if i == 0 {
					categoryOutputFilePath = categoryOutputBasePath + "/" + INDEX_HTML
				} else {
					basePath := categoryOutputBasePath + "/page/" + strconv.Itoa(i+1)
					os.MkdirAll(basePath, 0755)
					categoryOutputFilePath = basePath + "/" + INDEX_HTML
				}
				indexOutputFile, err := os.Create(categoryOutputFilePath)
				if err != nil {
					log.Panic(err)
				}
				defer indexOutputFile.Close()

				if err := indexTemplate.Execute(indexOutputFile, categoryArticlesPart); err != nil {
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

	// goroutine外でカテゴリスライスから記事がないカテゴリをまとめて削除
	if len(noArticleCategoryIDs) > 0 {
		filteredCategories := make([]Category, 0, len(categoriesList.Categories))
		noArticleCategoryIDSet := make(map[string]struct{}, len(noArticleCategoryIDs))
		for _, id := range noArticleCategoryIDs {
			noArticleCategoryIDSet[id] = struct{}{}
		}
		for _, cat := range categoriesList.Categories {
			if _, found := noArticleCategoryIDSet[cat.ID]; !found {
				filteredCategories = append(filteredCategories, cat)
			}
		}
		categoriesList.Categories = filteredCategories
	}

	// カテゴリのJSONを保存
	categoriesFile, err := os.Create(cfg.Paths.ExportPath + "/category.json")
	if err != nil {
		log.Panic(err)
	}
	defer categoriesFile.Close()

	x, err := json.Marshal(categoriesList.Categories)
	if err != nil {
		log.Panic(err)
	}
	categoriesFile.WriteString(string(x))

	// ----------------
	// Singlesページ生成
	// ----------------
	log.Print(">> Rendering singles pages")
	// Singlesページの描画
	var singleTemplates []string

	if err := filepath.WalkDir(cfg.Paths.SinglesTemplatesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		singleTemplates = append(singleTemplates, path)
		return nil
	}); err != nil {
		log.Panic(err)
	}

	var wgSingles sync.WaitGroup
	singlesCounter := 0
	for _, tmplPath := range singleTemplates {
		relPath, err := filepath.Rel(cfg.Paths.SinglesTemplatesPath, tmplPath)
		if err != nil {
			log.Panic(err)
		}

		wgSingles.Add(1)
		go func(tmplPath, relPath string) {
			plusSingle := append([]string{tmplPath}, componentFilesName...)
			singleTemplate := template.Must(template.New(filepath.Base(tmplPath)).Funcs(functionMapping).ParseFiles(plusSingle...))

			outputFilePath := filepath.Join(cfg.Paths.ExportPath, relPath)
			os.MkdirAll(filepath.Dir(outputFilePath), 0755)
			outFile, err := os.Create(outputFilePath)
			if err != nil {
				log.Panic(err)
			}
			defer outFile.Close()

			data := struct {
				Latest     []Article
				Categories []Category
			}{articlesLatest.Articles, categoriesList.Categories}

			if err := singleTemplate.Execute(outFile, data); err != nil {
				log.Panic(err)
			}

			mu.Lock()
			singlesCounter++
			log.Print("Rendered single ", singlesCounter, " / ", len(singleTemplates), " '", relPath, "'")
			mu.Unlock()
			wgSingles.Done()
		}(tmplPath, relPath)
	}
	wgSingles.Wait()

	log.Print("Rendering done!")
}
