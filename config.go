package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

const (
	DEFAULT_PAGE_SHOW_LIMIT = 10
	DEFAULT_LATEST_ARTICLES = 5

	DEFAULT_EXPORT_PATH            = "./output"
	DEFAULT_TEMPLATES_PATH         = "./templates"
	DEFAULT_COMPONENTS_PATH        = "./templates/components"
	DEFAULT_BLOG_TEMPLATES_PATH    = "./templates/blog"
	DEFAULT_SINGLES_TEMPLATES_PATH = "./templates/singles"
	DEFAULT_STATIC_PATH            = "./static"

	DEFAULT_CATEGORY_TAG_NAME = "Category"
	DEFAULT_TIME_ARCHIVE_NAME = "Archive"
)

type MagicNumber struct {
	FreeContentsLimit int
}

type Paths struct {
	ExportPath           string
	TemplatesPath        string
	ComponentsPath       string
	BlogTemplatesPath    string
	SinglesTemplatesPath string
	StaticPath           string
}

type Config struct {
	Apikey          string
	ServiceDomain   string
	PageShowLimit   int
	Timezone        string
	CategoryTagName string
	TimeArchiveName string
	Tz              *time.Location

	Paths          Paths
	LatestArticles int
	MgNum          MagicNumber
}

func getEnvOrFatal(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("Error: Environment variable '%s' not found.", key)
	}
	return value
}

// ジェネリクス対応のgetEnvOrDefault
func getEnvOrDefault[T any](key string, def T) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	var result any
	switch any(def).(type) {
	case int:
		v, err := strconv.Atoi(value)
		if err != nil {
			return def
		}
		result = v
	case bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return def
		}
		result = v
	case float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return def
		}
		result = v
	case string:
		return any(value).(T)
	default:
		return def
	}
	return result.(T)
}

func LoadConfig() (Config, error) {
	var Config Config
	var Paths Paths

	log.Print("Loading the setting values from environment variables")

	Config.Apikey = getEnvOrFatal("MICROCMS_API_KEY")
	Config.ServiceDomain = getEnvOrFatal("SERVICE_DOMAIN")

	Paths.ExportPath = getEnvOrDefault("EXPORT_PATH", DEFAULT_EXPORT_PATH)
	Paths.TemplatesPath = getEnvOrDefault("TEMPLATES_PATH", DEFAULT_TEMPLATES_PATH)
	Paths.ComponentsPath = getEnvOrDefault("COMPONENTS_PATH", DEFAULT_COMPONENTS_PATH)
	Paths.BlogTemplatesPath = getEnvOrDefault("BLOG_TEMPLATES_PATH", DEFAULT_BLOG_TEMPLATES_PATH)
	Paths.SinglesTemplatesPath = getEnvOrDefault("SINGLES_TEMPLATES_PATH", DEFAULT_SINGLES_TEMPLATES_PATH)
	Paths.StaticPath = getEnvOrDefault("STATIC_PATH", DEFAULT_STATIC_PATH)

	PageShowLimit := getEnvOrDefault("PAGE_SHOW_LIMIT", DEFAULT_PAGE_SHOW_LIMIT)
	if PageShowLimit <= 0 {
		Config.PageShowLimit = DEFAULT_PAGE_SHOW_LIMIT
	} else {
		Config.PageShowLimit = PageShowLimit
	}

	Config.Timezone = getEnvOrDefault("TIMEZONE", "UTC")
	Config.CategoryTagName = getEnvOrDefault("CATEGORY_TAG_NAME", DEFAULT_CATEGORY_TAG_NAME)
	Config.TimeArchiveName = getEnvOrDefault("TIME_ARCHIVE_NAME", DEFAULT_TIME_ARCHIVE_NAME)
	Config.LatestArticles = getEnvOrDefault("LATEST_ARTICLES", DEFAULT_LATEST_ARTICLES)
	Config.MgNum = MagicNumber{
		FreeContentsLimit: 10000,
	}

	tz, err := time.LoadLocation(Config.Timezone)
	if err != nil {
		log.Fatal("Error: Invalid timezone: " + Config.Timezone)
	}

	Config.Paths = Paths

	// 設定値出力
	log.Print("Configuration values:")
	log.Print("AssetsDirName: " + Config.Paths.StaticPath)
	log.Print("Exportpath: " + Config.Paths.ExportPath)
	log.Print("PageShowLimit: " + strconv.Itoa(Config.PageShowLimit))
	log.Print("Templatepath: " + Config.Paths.TemplatesPath)
	log.Print("Timezone: " + Config.Timezone)
	log.Print("---------------")

	Config.Tz = tz
	return Config, nil
}
