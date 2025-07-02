package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"text/template"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/microcmsio/microcms-go-sdk"
)

// renderMainPagesAndArticles はメインページと記事ページをレンダリングします。
func renderMainPagesAndArticles(
	cfg Config,
	client *microcms.Client,
	indexTemplate *template.Template,
	articleTemplate *template.Template,
	articlesLatest ArticleList,
	actualPageCount int,
	pageLimit int,
	componentFilesName []string, // Singlesページレンダリング時に利用
	functionMapping template.FuncMap, // Singlesページレンダリング時に利用
) {
	var mu sync.Mutex
	articleCounter := 0
	totalArticles := articlesLatest.Totalcount // 総記事数を保持

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
		articlesPart.PrevPage = i
		articlesPart.AllPage = actualPageCount
		articlesPart.Root = "/"
		articlesPart.IsIndex = true
		articlesPart.Totalcount = totalArticles // 総記事数を ArticleList にも設定

		var outputFilePath string
		if i == 0 {
			outputFilePath = cfg.Paths.ExportPath + "/index.html"
		} else {
			outputBasePath := cfg.Paths.ExportPath + "/page/" + strconv.Itoa(i+1)
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

		var wgArticles sync.WaitGroup
		for a := 0; a < len(articlesPart.Articles); a++ {
			wgArticles.Add(1)
			go func(article Article) {
				defer wgArticles.Done()
				outputFilePath := cfg.Paths.ExportPath + "/articles/" + article.ID + ".html"
				articleOutputFile, err := os.Create(outputFilePath)
				if err != nil {
					log.Panic(err)
				}
				defer articleOutputFile.Close()

				if err := articleTemplate.Execute(articleOutputFile, article); err != nil {
					log.Panic(err)
				}

				mu.Lock()
				articleCounter++
				log.Print("- Rendered articles ", articleCounter, " / ", totalArticles)
				mu.Unlock()
			}(articlesPart.Articles[a])
		}
		wgArticles.Wait()
	}
}

// renderCategoryPages はカテゴリページをレンダリングします。
func renderCategoryPages(
	cfg Config,
	client *microcms.Client,
	indexTemplate *template.Template,
	categoriesList CategoryList,
	pageLimit int,
) {
	var mu sync.Mutex
	categories := categoriesList.Categories
	var wgCategories sync.WaitGroup
	categoryCounter := 0

	for c := 0; c < len(categories); c++ {
		wgCategories.Add(1)
		go func(category Category) {
			defer wgCategories.Done()
			var categoryArticlesMinimum ArticleList
			categoryID := category.ID

			if err := client.List(
				microcms.ListParams{
					Endpoint: "article",
					Fields:   []string{"id"},
					Limit:    cfg.MgNum.FreeContentsLimit, // Note: config.goで定義されている想定
					Orders:   []string{"-publishedAt"},
					Filters:  "category[contains]" + categoryID,
				}, &categoryArticlesMinimum,
			); err != nil {
				log.Panic(err)
			}

			contentsCount := categoryArticlesMinimum.Totalcount
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
						Orders:   []string{"-publishedAt"}, // 記事の並び順を考慮
					}, &categoryArticlesPart,
				); err != nil {
					log.Panic(err)
				}

				categoryArticlesPart.NextPage = i + 2
				categoryArticlesPart.PrevPage = i
				categoryArticlesPart.AllPage = loopsCount
				categoryArticlesPart.Root = "/articles/category/" + categoryID + "/"
				categoryArticlesPart.IsIndex = false
				categoryArticlesPart.ArchiveName = cfg.CategoryTagName + ": " + category.Name
				categoryArticlesPart.Totalcount = contentsCount // カテゴリ内の総記事数を設定

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

				if err := indexTemplate.Execute(indexOutputFile, categoryArticlesPart); err != nil {
					log.Panic(err)
				}
			}

			mu.Lock()
			categoryCounter++
			log.Print("Rendered category ", categoryCounter, " / ", len(categories), " '"+categoryID+"'")
			mu.Unlock()
		}(categories[c])
	}
	wgCategories.Wait()
}

// renderSinglePages はSinglesページをレンダリングします。
func renderSinglePages(
	cfg Config,
	articlesLatest ArticleList, // main.go から渡す
	categoriesList CategoryList, // main.go から渡す
	componentFilesName []string,
	functionMapping template.FuncMap, // main.goのヘルパー関数マッピング
) {
	log.Print(">> Rendering singles pages")
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
	var mu sync.Mutex
	singlesCounter := 0
	for _, tmplPath := range singleTemplates {
		relPath, err := filepath.Rel(cfg.Paths.SinglesTemplatesPath, tmplPath)
		if err != nil {
			log.Panic(err)
		}

		wgSingles.Add(1)
		go func(tmplPath, relPath string) {
			defer wgSingles.Done()
			// Singlesページごとにテンプレートをパース
			plusSingle := append([]string{tmplPath}, componentFilesName...)
			singleTemplate := template.Must(template.New(filepath.Base(tmplPath)).Funcs(functionMapping).ParseFiles(plusSingle...))

			outputFilePath := filepath.Join(cfg.Paths.ExportPath, relPath)
			os.MkdirAll(filepath.Dir(outputFilePath), 0755) // Ensure directory exists
			outFile, err := os.Create(outputFilePath)
			if err != nil {
				log.Panic(err)
			}
			defer outFile.Close()

			// Singlesページに渡すデータ構造を調整
			data := struct {
				Latest     []Article
				Categories []Category
				// 必要であれば他のフィールドも追加
			}{
				Latest:     articlesLatest.Articles,
				Categories: categoriesList.Categories,
			}

			if err := singleTemplate.Execute(outFile, data); err != nil {
				log.Panic(err)
			}
			mu.Lock()
			singlesCounter++
			log.Print("Rendered single ", singlesCounter, " / ", len(singleTemplates), " '", relPath, "'")
			mu.Unlock()
		}(tmplPath, relPath)
	}
	wgSingles.Wait()
}

// saveLatestArticlesJSON は最新記事のJSONを保存します。
func saveLatestArticlesJSON(cfg Config, articlesLatest ArticleList) {
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
}

// saveCategoriesJSON はカテゴリのJSONを保存します。
func saveCategoriesJSON(cfg Config, categoriesList CategoryList) {
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
}
