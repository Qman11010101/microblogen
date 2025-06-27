package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

type ConfigStruct struct {
	Apikey          string
	Servicedomain   string
	Exportpath      string
	Templatepath    string
	AssetsDirName   string
	PageShowLimit   int
	Timezone        string
	CategoryTagName string
	TimeArchiveName string
	Tz              *time.Location
}

func LoadConfig() (ConfigStruct, error) {
	// --------------
	// 環境変数読み込み
	// --------------
	var Config ConfigStruct

	log.Print("Loading the setting values from environment variables")

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

	Config.Tz = tz
	return Config, nil
}
