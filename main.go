package main

import (
	"encoding/json"
	"log"
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
const copyAssetsFile = "copyassets.json"
const VERSION = "0.2"

type ConfigStruct struct {
	Apikey        string `json:"APIkey"`
	Servicedomain string `json:"serviceDomain"`
	Exportpath    string `json:"exportPath"`
	Templatepath  string `json:"templatePath"`
	AssetsDirName string `json:"assetsDirName"`
	PageShowLimit int    `json:"pageShowLimit"`
}
type CopyingAssets struct {
	Assets []string `json:"assets"`
}

type ArticleList struct {
	Articles   []Article `json:"contents"`
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
}
type CategoryList struct {
	Categories []Category `json:"contents,omitempty"`
	Totalcount int        `json:"totalCount,omitempty"`
	Offset     int        `json:"offset,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}

var Config ConfigStruct
var CopyAssets CopyingAssets

// Utility Function

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
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
	if fileExists(configFile) {
		configFileBytes, err := os.ReadFile(configFile)
		if err != nil {
			log.Panic(err)
		}

		err = json.Unmarshal([]byte(configFileBytes), &Config)
		if err != nil {
			log.Panic(err)
		}
	} else {
		log.Print(configFile, " not found. Loading the setting values from environment variables instead")
		Apikey, ok := os.LookupEnv("MICROCMS_API_KEY")
		if ok {
			Config.Apikey = Apikey
		} else {
			log.Fatal("Error: Environment variable 'MICROCMS_API_KEY' not found.")
		}
		Servicedomain, ok := os.LookupEnv("SERVICE_DOMAIN")
		if ok {
			Config.Servicedomain = Servicedomain
		} else {
			log.Fatal("Error: Environment variable 'SERVICE_DOMAIN' not found.")
		}
		Exportpath, ok := os.LookupEnv("EXPORT_PATH")
		if ok {
			Config.Exportpath = Exportpath
		} else {
			Config.Exportpath = "./output"
		}
		Templatepath, ok := os.LookupEnv("TEMPLATE_PATH")
		if ok {
			Config.Templatepath = Templatepath
		} else {
			Config.Templatepath = "./template"
		}
		PageShowLimit, ok := os.LookupEnv("PAGE_SHOW_LIMIT")
		if ok {
			value, err := strconv.Atoi(PageShowLimit)
			if err != nil {
				log.Print("Warning: Environment variable 'PAGE_SHOW_LIMIT' is not integer; Use default value.")
				Config.PageShowLimit = 10
			} else {
				Config.PageShowLimit = value
			}
		} else {
			Config.PageShowLimit = 10
		}
	}

	// -----------------
	// テンプレート存在確認
	// -----------------
	if !fileExists(Config.Templatepath) || !fileExists(Config.Templatepath+"/article.html") || !fileExists(Config.Templatepath+"/index.html") {
		log.Fatal("Error: Missing templates. You must prepare \"article.html\" and \"index.html\" inside ./" + Config.Templatepath + ".")
	}

	// ---------------
	// 出力フォルダ再生成
	// ---------------
	if fileExists(Config.Exportpath) {
		log.Print(">> Removing existing export directory")
		if err := os.RemoveAll(Config.Exportpath); err != nil {
			log.Panic(err)
		}
	}

	log.Print(">> Generating export directory")
	os.MkdirAll(Config.Exportpath+"/articles/category/", 0777)

	// ------------
	// アセットコピー
	// ------------
	if fileExists(copyAssetsFile) {
		copyAssetsFileBytes, err := os.ReadFile(copyAssetsFile)
		if err != nil {
			log.Panic(err)
		}

		err = json.Unmarshal([]byte(copyAssetsFileBytes), &CopyAssets)
		if err != nil {
			log.Panic(err)
		}

		log.Print(">> Copying assets")

		for i := 0; i < len(CopyAssets.Assets); i++ {
			assetObjName := CopyAssets.Assets[i]
			if fileExists(Config.Templatepath + "/" + assetObjName) {
				log.Print(">>>> Copying " + assetObjName)
				copy.Copy(Config.Templatepath+"/"+assetObjName, Config.Exportpath+"/"+assetObjName)
			} else {
				log.Print("Warning: " + assetObjName + " does not exist; Skipped!")
			}
		}
	} else {
		log.Print("Warning: " + copyAssetsFile + "not found; No assets will be copied. Please prepare '" + copyAssetsFile + "' if you want to copy assets.")
	}

	// microcms用クライアントインスタンス生成
	client := microcms.New(Config.Servicedomain, Config.Apikey)

	// 先にミニマムなlatest用のやつ落としてcontent数(totalCount)を取得できるようにしておく
	var articlesLatest ArticleList

	err := client.List(
		microcms.ListParams{
			Endpoint: "article",
			Fields:   []string{"id", "title", "publishedAt", "updatedAt", "category.id", "category.name"},
			Limit:    5,
			Orders:   []string{"-publishedAt"},
		}, &articlesLatest)

	if err != nil {
		log.Panic(err)
	}

	// 最新記事のJSONを保存
	articlesFile, err := os.Create(Config.Exportpath + "/latest.json")
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
	pageLimit := Config.PageShowLimit
	loopsCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

	// -----------------------------------
	// メインページ(index.html)/記事ページ生成
	// -----------------------------------

	// HTMLタグ消去用正規表現
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

	log.Print(">> Rendering start ")
	for i := 0; i < loopsCount; i++ {
		log.Print("Rendering mainpage ", i+1, " / ", loopsCount)
		var articlesPart ArticleList

		err := client.List(
			microcms.ListParams{
				Endpoint: "article",
				Fields:   []string{"id", "title", "body", "publishedAt", "updatedAt", "category.id", "category.name"},
				Limit:    pageLimit,
				Offset:   pageLimit * i,
				Orders:   []string{"-publishedAt"},
			}, &articlesPart)

		if err != nil {
			log.Panic(err)
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
			log.Panic(err)
		}
		defer indexOutputFile.Close()

		if err := indexTemplate.Execute(indexOutputFile, articlesPart); err != nil {
			log.Panic(err)
		}

		// 記事レンダリング
		for a := 0; a < len(articlesPart.Articles); a++ {
			log.Print("- Rendering articles ", pageLimit*i+a+1, " / ", articlesPart.Totalcount)
			articleTemplate := template.Must(template.New("article.html").Funcs(functionMapping).ParseFiles(Config.Templatepath + "/article.html"))
			outputFilePath := Config.Exportpath + "/articles/" + articlesPart.Articles[a].ID + ".html"
			articleOutputFile, err := os.Create(outputFilePath)
			if err != nil {
				log.Panic(err)
			}
			defer articleOutputFile.Close()

			if err := articleTemplate.Execute(articleOutputFile, articlesPart.Articles[a]); err != nil {
				log.Panic(err)
			}
		}
	}

	// ---------------
	// カテゴリページ生成
	// ---------------

	// カテゴリ(タグ)の構造体
	var categoriesList CategoryList

	err = client.List(
		microcms.ListParams{
			Endpoint: "category",
			Fields:   []string{"id", "name"},
			Limit:    10000, // 無料枠リミット
		},
		&categoriesList,
	)
	if err != nil {
		log.Panic(err)
	}

	// カテゴリのJSONを保存
	categoriesFile, err := os.Create(Config.Exportpath + "/category.json")
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
	for c := 0; c < len(categories); c++ {
		var categoryArticlesMinimum ArticleList
		categoryID := categories[c].ID
		log.Print("Rendering category ", c+1, " / ", len(categories), " '"+categoryID+"'")

		err := client.List(
			microcms.ListParams{
				Endpoint: "article",
				Fields:   []string{"id"},
				Limit:    10000, // 無料枠リミット
				Orders:   []string{"-publishedAt"},
				Filters:  "category[contains]" + categoryID,
			}, &categoryArticlesMinimum)

		if err != nil {
			log.Panic(err)
		}

		contentsCount := categoryArticlesMinimum.Totalcount
		loopsCount := int(math.Ceil(float64(contentsCount) / float64(pageLimit)))

		categoryOutputBasePath := Config.Exportpath + "/articles/category/" + categoryID
		os.MkdirAll(categoryOutputBasePath, 0755)

		for i := 0; i < loopsCount; i++ {
			var categoryArticlesPart ArticleList

			err := client.List(
				microcms.ListParams{
					Endpoint: "article",
					Fields:   []string{"id", "title", "body", "publishedAt", "updatedAt", "category.id", "category.name"},
					Limit:    pageLimit,
					Offset:   pageLimit * i,
					Filters:  "category[contains]" + categoryID,
				}, &categoryArticlesPart)

			if err != nil {
				log.Panic(err)
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
				log.Panic(err)
			}
			defer indexOutputFile.Close()

			if err := categoryIndexTemplate.Execute(indexOutputFile, categoryArticlesPart); err != nil {
				log.Panic(err)
			}
		}
	}

	log.Print("Rendering done!")
}
