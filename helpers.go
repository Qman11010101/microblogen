package main

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// 
var re = regexp.MustCompile(`<img[^>]*\bsrc\s*=\s*['"]?([^'">]+)['"]?[^>]*>`)

// HelperContext contains the context for helper functions, such as timezone.
type HelperContext struct {
	Tz *time.Location
}

func HelperFunctionsMapping(ctx HelperContext) template.FuncMap {
	// htmlTagTrimReg is a regex to remove HTML tags from a string.
	htmlTagTrimReg := regexp.MustCompile(`<[^>]*>`)

	return template.FuncMap{
		"formatTime":   func(t time.Time) string { return t.In(ctx.Tz).Format("2006-01-02") },
		"totalGreater": func(total, limit int) bool { return total > limit },
		"isNotFirst":   func(offset int) bool { return offset != 0 },
		"isNotLast":    func(limit, offset, total int) bool { return limit+offset < total },
		"trimSample": func(body string) string {
			r := []rune(htmlTagTrimReg.ReplaceAllString(body, ""))
			return string(r[:int(math.Min(100, float64(len(r))))]) + "…"
		},
		"sub":           func(a, b int) int { return a - b },
		"replaceWebp":   func(body string) string { return convertWebp(body) },
		"buildTime":     func() string { return strconv.FormatInt(time.Now().Unix(), 10) },
		"getPagination": getPagination,
	}
}

func getPagination(current, allCount, pageRange int) []int {
	if allCount <= 0 || pageRange <= 0 {
		return []int{}
	}
	if pageRange > allCount {
		pageRange = allCount
	}

	half := pageRange / 2
	start := current - half
	end := current + half

	// 左端に寄せる場合
	if start < 1 {
		start = 1
		end = start + pageRange - 1
	}
	// 右端に寄せる場合
	if end > allCount {
		end = allCount
		start = end - pageRange + 1
		if start < 1 {
			start = 1
		}
	}

	result := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		result = append(result, i)
	}
	return result
}

func convertWebp(html string) string {

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
