package main

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func HelperFunctionsMapping(tz *time.Location) template.FuncMap {
	// htmlTagTrimReg is a regex to remove HTML tags from a string.
	htmlTagTrimReg := regexp.MustCompile(`<[^>]*>`)

	return template.FuncMap{
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
