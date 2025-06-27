package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"
)

const configFile = "config.json"

// Magic numbers
const (
	DEFAULT_PAGE_SHOW_LIMIT = 10
	DEFAULT_LATEST_ARTICLES = 5
	FREE_CONTENTS_LIMIT     = 10000
)

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

var Cfg ConfigStruct

// FileExists checks if a file or directory exists.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// LoadConfig loads configuration from config.json or environment variables.
func LoadConfig() {
	if FileExists(configFile) {
		configFileBytes, err := os.ReadFile(configFile)
		if err != nil {
			log.Panic(err)
		}

		err = json.Unmarshal([]byte(configFileBytes), &Cfg)
		if err != nil {
			log.Panic(err)
		}

		if Cfg.PageShowLimit <= 0 {
			log.Printf("Warning: pageShowLimit from config.json is %d (non-positive); using default %d", Cfg.PageShowLimit, DEFAULT_PAGE_SHOW_LIMIT)
			Cfg.PageShowLimit = DEFAULT_PAGE_SHOW_LIMIT
		}

		// Timezone未設定ならUTC
		if Cfg.Timezone == "" {
			Cfg.Timezone = "UTC"
		}
	} else {
		log.Print(configFile, " not found. Loading the setting values from environment variables instead")
		Apikey, ok := os.LookupEnv("MICROCMS_API_KEY")
		if ok {
			Cfg.Apikey = Apikey
		} else {
			log.Fatal("Error: Environment variable 'MICROCMS_API_KEY' not found.")
		}
		Servicedomain, ok := os.LookupEnv("SERVICE_DOMAIN")
		if ok {
			Cfg.Servicedomain = Servicedomain
		} else {
			log.Fatal("Error: Environment variable 'SERVICE_DOMAIN' not found.")
		}
		Exportpath, ok := os.LookupEnv("EXPORT_PATH")
		if ok {
			Cfg.Exportpath = Exportpath
		} else {
			Cfg.Exportpath = "./output"
		}
		Templatepath, ok := os.LookupEnv("TEMPLATE_PATH")
		if ok {
			Cfg.Templatepath = Templatepath
		} else {
			Cfg.Templatepath = "./template"
		}
		PageShowLimit, ok := os.LookupEnv("PAGE_SHOW_LIMIT")
		if ok {
			value, err := strconv.Atoi(PageShowLimit)
			if err != nil || value <= 0 {
				log.Printf("Warning: Environment variable 'PAGE_SHOW_LIMIT' is '%s' which is not a positive integer; Using default value %d.", PageShowLimit, DEFAULT_PAGE_SHOW_LIMIT)
				Cfg.PageShowLimit = DEFAULT_PAGE_SHOW_LIMIT
			} else {
				Cfg.PageShowLimit = value
			}
		} else {
			Cfg.PageShowLimit = DEFAULT_PAGE_SHOW_LIMIT
		}
		Timezone, ok := os.LookupEnv("TIMEZONE")
		if ok {
			Cfg.Timezone = Timezone
		} else {
			Cfg.Timezone = "UTC"
		}
		CategoryTagName, ok := os.LookupEnv("CATEGORY_TAG_NAME")
		if ok {
			Cfg.CategoryTagName = CategoryTagName
		} else {
			Cfg.CategoryTagName = "Category"
		}
		TimeArchiveName, ok := os.LookupEnv("TIME_ARCHIVE_NAME")
		if ok {
			Cfg.TimeArchiveName = TimeArchiveName
		} else {
			Cfg.TimeArchiveName = "Archive"
		}
	}

	_, err := time.LoadLocation(Cfg.Timezone)
	if err != nil {
		log.Fatal("Error: Invalid timezone: " + Cfg.Timezone)
	}

	// 設定値出力
	log.Print("Configuration values:")
	log.Print("AssetsDirName: " + Cfg.AssetsDirName)
	log.Print("Exportpath: " + Cfg.Exportpath)
	log.Print("PageShowLimit: " + strconv.Itoa(Cfg.PageShowLimit))
	log.Print("Templatepath: " + Cfg.Templatepath)
	log.Print("Timezone: " + Cfg.Timezone)
	log.Print("---------------")
}
