package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

const (
	DEFAULT_ARTICLES_PER_PAGE = 10
	DEFAULT_LATEST_ARTICLES   = 5

	DEFAULT_EXPORT_PATH    = "./output"
	DEFAULT_RESOURCES_PATH = "./resources"

	DEFAULT_CATEGORY_TAG_NAME = "Category"
	DEFAULT_TIME_ARCHIVE_NAME = "Archive"

	STATIC_DIR_NAME            = "/static"
	TEMPLATES_DIR_NAME         = "/templates"
	COMPONENTS_DIR_NAME        = "/components"
	BLOG_TEMPLATES_DIR_NAME    = "/blog"
	SINGLES_TEMPLATES_DIR_NAME = "/singles"
)

type Paths struct {
	// Configurable paths
	ExportPath    string
	ResourcesPath string

	StaticPath           string
	TemplatesPath        string
	ComponentsPath       string
	BlogTemplatesPath    string
	SinglesTemplatesPath string
}

type Config struct {
	Apikey          string
	ServiceDomain   string
	ArticlesPerPage int
	Timezone        string
	CategoryTagName string
	TimeArchiveName string
	Tz              *time.Location

	Paths          Paths
	LatestArticles int
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

	Config.Apikey = getEnvOrFatal("MICROCMS_API_KEY")
	Config.ServiceDomain = getEnvOrFatal("SERVICE_DOMAIN")

	Paths.ExportPath = getEnvOrDefault("EXPORT_PATH", DEFAULT_EXPORT_PATH)
	Paths.ResourcesPath = getEnvOrDefault("RESOURCES_PATH", DEFAULT_RESOURCES_PATH)

	Paths.StaticPath = Paths.ResourcesPath + STATIC_DIR_NAME
	Paths.TemplatesPath = Paths.ResourcesPath + TEMPLATES_DIR_NAME
	Paths.ComponentsPath = Paths.TemplatesPath + COMPONENTS_DIR_NAME
	Paths.BlogTemplatesPath = Paths.TemplatesPath + BLOG_TEMPLATES_DIR_NAME
	Paths.SinglesTemplatesPath = Paths.TemplatesPath + SINGLES_TEMPLATES_DIR_NAME

	articlesPerPage := getEnvOrDefault("ARTICLES_PER_PAGE", DEFAULT_ARTICLES_PER_PAGE)
	if articlesPerPage <= 0 {
		Config.ArticlesPerPage = DEFAULT_ARTICLES_PER_PAGE
	} else {
		Config.ArticlesPerPage = articlesPerPage
	}

	Config.Timezone = getEnvOrDefault("TIMEZONE", "UTC")
	Config.CategoryTagName = getEnvOrDefault("CATEGORY_TAG_NAME", DEFAULT_CATEGORY_TAG_NAME)
	Config.TimeArchiveName = getEnvOrDefault("TIME_ARCHIVE_NAME", DEFAULT_TIME_ARCHIVE_NAME)
	Config.LatestArticles = getEnvOrDefault("LATEST_ARTICLES", DEFAULT_LATEST_ARTICLES)

	tz, err := time.LoadLocation(Config.Timezone)
	if err != nil {
		log.Fatal("Error: Invalid timezone: " + Config.Timezone)
	}

	Config.Paths = Paths
	Config.Tz = tz
	return Config, nil
}
