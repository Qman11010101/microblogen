package main

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/microcmsio/microcms-go-sdk"
)

const configFile = "config.json"
const copyAssetsFile = "copyassets.json"
const VERSION = "1.7.0"

const componentsDirPath = "/components"

type ConfigStruct struct {
	Apikey          string `json:"APIkey"`
	Servicedomain   string `json:"serviceDomain"`
	Exportpath      string `json:"exportPath"`
	Templatepath    string `json:"templatePath"`
	AssetsDirName   string `json:"assetsDirName"`
	PageShowLimit   int    `json:"pageShowLimit"`
	Timezone        string `json:"timezone"`
	CategoryTagName string `json:"categoryTagName"`
	TimeArchiveName string `json:"timeArchiveName"`
}

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

var Config ConfigStruct

// Magic numbers
const (
	DEFAULT_PAGE_SHOW_LIMIT = 10
	DEFAULT_LATEST_ARTICLES = 5
	FREE_CONTENTS_LIMIT     = 10000
)

// Utility Function
func fileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func convertWebp(html string) string {
	// HTMLからimgタグのsrc属性を抽出するための正規表現
	re := regexp.MustCompile(`<img[^>]*\bsrc\s*=\s*['"]?([^'">]+)['"]?[^>]*>`)

	// 正規表現を使ってimgタグのsrc属性を抽出し、条件に合致するURLに"?fm=webp"を付加して置換する
	convertedHTML := re.ReplaceAllStringFunc(html, func(match string) string {
		url := re.FindStringSubmatch(match)[1]

		if strings.HasPrefix(url, "https://images.microcms-assets.io/assets/") && (strings.HasSuffix(url, ".jpg") || strings.HasSuffix(url, ".png")) {
			return strings.ReplaceAll(match, url, url+"?fm=webp")
		}

		return match
	})

	return convertedHTML
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

		if Config.PageShowLimit <= 0 {
			log.Printf("Warning: pageShowLimit from config.json is %d (non-positive); using default %d", Config.PageShowLimit, DEFAULT_PAGE_SHOW_LIMIT)
			Config.PageShowLimit = DEFAULT_PAGE_SHOW_LIMIT
		}

		// Timezone未設定ならUTC
		if Config.Timezone == "" {
			Config.Timezone = "UTC"
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
			if err != nil || value <= 0 {
				log.Printf("Warning: Environment variable 'PAGE_SHOW_LIMIT' is '%s' which is not a positive integer; Using default value %d.", PageShowLimit, DEFAULT_PAGE_SHOW_LIMIT)
				Config.PageShowLimit = DEFAULT_PAGE_SHOW_LIMIT
			} else {
				Config.PageShowLimit = value
			}
		} else {
			Config.PageShowLimit = DEFAULT_PAGE_SHOW_LIMIT
		}
		Timezone, ok := os.LookupEnv("TIMEZONE")
		if ok {
			Config.Timezone = Timezone
		} else {
			Config.Timezone = "UTC"
		}
		CategoryTagName, ok := os.LookupEnv("CATEGORY_TAG_NAME")
		if ok {
			Config.CategoryTagName = CategoryTagName
		} else {
			Config.CategoryTagName = "Category"
		}
		TimeArchiveName, ok := os.LookupEnv("TIME_ARCHIVE_NAME")
		if ok {
			Config.TimeArchiveName = TimeArchiveName
		} else {
			Config.TimeArchiveName = "Archive"
		}
	}

	tz, err := time.LoadLocation(Config.Timezone)
	if err != nil {
		log.Fatal("Error: Invalid timezone: " + Config.Timezone)
	}

	// 設定値出力
	log.Print("Configuration values:")
	log.Print("AssetsDirName: " + Config.AssetsDirName)
	log.Print("Exportpath: " + Config.Exportpath)
	log.Print("PageShowLimit: " + strconv.Itoa(Config.PageShowLimit))
	log.Print("Templatepath: " + Config.Templatepath)
	log.Print("Timezone: " + Config.Timezone)
	log.Print("---------------")

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

	// コンポーネント一覧を取得(なければディレクトリだけ生成)
	var componentDirPath = Config.Templatepath + componentsDirPath
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
	os.MkdirAll(Config.Exportpath+"/articles/category/", 0777)

	// TODO
	// .mbignoreに記載されたファイル：除外
	// 記載されていないhtmlファイル：レンダリング（レンダリングされる部分がない場合はそのファイルがコピーされる挙動にする）
	// 記載されていない非htmlファイル：コピー（画像やCSSなど）
	// config.jsonにテンプレートの拡張子を指定できるようにしてもいいかも　デフォルトはhtml
	// componentsディレクトリも除外対象ではあるので、ここらへんをもう少しわかりやすい形にしたい

	// microcms用クライアントインスタンス生成
	client := microcms.New(Config.Servicedomain, Config.Apikey)

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
		"formatTime":   func(t time.Time) string { return t.In(tz).Format("2006-01-02") },
		"totalGreater": func(total, limit int) bool { return total > limit },
		"isNotFirst":   func(offset int) bool { return offset != 0 },
		"isNotLast":    func(limit, offset, total int) bool { return limit+offset < total },
		"trimSample": func(body string) string {
			r := []rune(htmlTagTrimReg.ReplaceAllString(body, ""))
			return string(r[:int(math.Min(100, float64(len(r))))]) + "…"
		},
		"sub":         func(a, b int) int { return a - b },
		"replaceWebp": func(body string) string { return convertWebp(body) },
		"buildTime":   func() string { return strconv.FormatInt(time.Now().Unix(), 10) },
	}

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
		plusIdx := append([]string{Config.Templatepath + "/index.html"}, componentFilesName...)
		indexTemplate := template.Must(template.New("index.html").Funcs(functionMapping).ParseFiles(plusIdx...))
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
		var wgArticles sync.WaitGroup

		for a := 0; a < len(articlesPart.Articles); a++ {
			wgArticles.Add(1)
			go func(i int, a int) {
				plusAtc := append([]string{Config.Templatepath + "/article.html"}, componentFilesName...)
				articleTemplate := template.Must(template.New("article.html").Funcs(functionMapping).ParseFiles(plusAtc...))
				outputFilePath := Config.Exportpath + "/articles/" + articlesPart.Articles[a].ID + ".html"
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

			categoryOutputBasePath := Config.Exportpath + "/articles/category/" + categoryID
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
				categoryArticlesPart.ArchiveName = Config.CategoryTagName + ": " + categories[c].Name

				// カテゴリのトップページ(index.html)レンダリング
				plusCatIdx := append([]string{Config.Templatepath + "/index.html"}, componentFilesName...)
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
