package main

import (
	"strings"
	"testing"
	"text/template"
	"time"
)

func TestFormatTime(t *testing.T) {
	locBerlin, _ := time.LoadLocation("Europe/Berlin")
	locTokyo, _ := time.LoadLocation("Asia/Tokyo")

	testCases := []struct {
		name     string
		inputTime time.Time
		tz       *time.Location
		expected string
	}{
		{
			name:     "UTC to UTC",
			inputTime: time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC),
			tz:       time.UTC,
			expected: "2023-10-26",
		},
		{
			name:     "UTC to Asia/Tokyo",
			inputTime: time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC), // 10:00 UTC is 19:00 JST
			tz:       locTokyo,
			expected: "2023-10-26",
		},
		{
			name:     "Asia/Tokyo to Europe/Berlin",
			inputTime: time.Date(2023, 10, 26, 19, 0, 0, 0, locTokyo), // 19:00 JST is 12:00 CEST (Berlin)
			tz:       locBerlin,
			expected: "2023-10-26",
		},
		{
			name:     "Across midnight",
			inputTime: time.Date(2023, 10, 26, 23, 0, 0, 0, locBerlin), // 23:00 Berlin (CEST) is 06:00 JST next day
			tz:       locTokyo,
			expected: "2023-10-27",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			helperCtx := HelperContext{Tz: tc.tz}
			funcMap := HelperFunctionsMapping(helperCtx)
			formatTimeFunc, ok := funcMap["formatTime"].(func(time.Time) string)
			if !ok {
				t.Fatal("formatTime function not found in FuncMap or has wrong type")
			}
			result := formatTimeFunc(tc.inputTime)
			if result != tc.expected {
				t.Errorf("Expected formatted time to be '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestTrimSample(t *testing.T) {
	helperCtx := HelperContext{} // Tz not needed for trimSample
	funcMap := HelperFunctionsMapping(helperCtx)
	trimSampleFunc, ok := funcMap["trimSample"].(func(string) string)
	if !ok {
		t.Fatal("trimSample function not found in FuncMap or has wrong type")
	}

	testCases := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "Simple HTML",
			body:     "<p>This is a <b>test</b> string.</p>",
			expected: "This is a test string.…",
		},
		{
			name:     "Long string",
			body:     "This is a very long string that definitely exceeds one hundred characters, so it should be truncated properly at the correct length to ensure the function works as expected.",
			expected: "This is a very long string that definitely exceeds one hundred characters, so it should be truncated p…",
		},
		{
			name:     "String shorter than 100 chars",
			body:     "Short string.",
			expected: "Short string.…",
		},
		{
			name:     "String with only HTML tags",
			body:     "<h1><a></a></h1><p></p>",
			expected: "…",
		},
		{
			name:     "Empty string",
			body:     "",
			expected: "…",
		},
		{
			name:     "String with multi-byte characters",
			body:     "<p>こんにちは世界。これはテストです。</p>This part is long enough to be cut. AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			expected: "こんにちは世界。これはテストです。This part is long enough to be cut. AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA…",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := trimSampleFunc(tc.body)
			if result != tc.expected {
				t.Errorf("Expected trimmed sample to be '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestConvertWebp(t *testing.T) {
	helperCtx := HelperContext{} // Tz not needed for convertWebp
	funcMap := HelperFunctionsMapping(helperCtx)
	replaceWebpFunc, ok := funcMap["replaceWebp"].(func(string) string)
	if !ok {
		t.Fatal("replaceWebp function not found in FuncMap or has wrong type")
	}

	testCases := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "JPG image from microCMS",
			html:     `<img src="https://images.microcms-assets.io/assets/service/endpoint/image.jpg">`,
			expected: `<img src="https://images.microcms-assets.io/assets/service/endpoint/image.jpg?fm=webp">`,
		},
		{
			name:     "PNG image from microCMS",
			html:     `<img src="https://images.microcms-assets.io/assets/service/endpoint/image.png" alt="test">`,
			expected: `<img src="https://images.microcms-assets.io/assets/service/endpoint/image.png?fm=webp" alt="test">`,
		},
		{
			name:     "Image not from microCMS",
			html:     `<img src="https://example.com/image.jpg">`,
			expected: `<img src="https://example.com/image.jpg">`,
		},
		{
			name:     "GIF image from microCMS (should not be converted)",
			html:     `<img src="https://images.microcms-assets.io/assets/service/endpoint/image.gif">`,
			expected: `<img src="https://images.microcms-assets.io/assets/service/endpoint/image.gif">`,
		},
		{
			name:     "Multiple images",
			html:     `<img src="https://images.microcms-assets.io/assets/1.jpg"><img src="https://example.com/2.png"><img src="https://images.microcms-assets.io/assets/3.png">`,
			expected: `<img src="https://images.microcms-assets.io/assets/1.jpg?fm=webp"><img src="https://example.com/2.png"><img src="https://images.microcms-assets.io/assets/3.png?fm=webp">`,
		},
		{
			name:     "Image with existing query parameters",
			html:     `<img src="https://images.microcms-assets.io/assets/service/endpoint/image.jpg?w=100&h=100">`,
			expected: `<img src="https://images.microcms-assets.io/assets/service/endpoint/image.jpg?fm=webp">`, // Current implementation replaces existing queries
		},
		{
			name:     "No img tags",
			html:     `<p>No images here.</p>`,
			expected: `<p>No images here.</p>`,
		},
		{
			name:     "Img tag with single quotes for src",
			html:     `<img src='https://images.microcms-assets.io/assets/service/endpoint/image.png'>`,
			expected: `<img src='https://images.microcms-assets.io/assets/service/endpoint/image.png?fm=webp'>`,
		},
		{
			name:     "Img tag with no quotes for src (less common but possible)",
			html:     `<img src=https://images.microcms-assets.io/assets/service/endpoint/image.png>`,
			expected: `<img src=https://images.microcms-assets.io/assets/service/endpoint/image.png?fm=webp>`,
		},
		{
			name:     "Img tag with extra spaces around src",
			html:     `<img  src = "https://images.microcms-assets.io/assets/service/endpoint/image.png" >`,
			expected: `<img  src = "https://images.microcms-assets.io/assets/service/endpoint/image.png?fm=webp" >`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := replaceWebpFunc(tc.html)
			if result != tc.expected {
				t.Errorf("Expected HTML '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGetTotalPages(t *testing.T) {
	helperCtx := HelperContext{} // Tz not needed
	funcMap := HelperFunctionsMapping(helperCtx)
	getTotalPagesFunc, ok := funcMap["getTotalPages"].(func(int, int) int)
	if !ok {
		t.Fatal("getTotalPages function not found in FuncMap or has wrong type")
	}

	testCases := []struct {
		name         string
		totalItems   int
		itemsPerPage int
		expected     int
	}{
		{"Exact division", 100, 10, 10},
		{"Remainder", 105, 10, 11},
		{"Fewer items than limit", 5, 10, 1},
		{"Zero items", 0, 10, 0},
		{"Zero items per page (should handle gracefully)", 100, 0, 0}, // Expect 0 or handle error if appropriate
		{"One item per page", 10, 1, 10},
		{"Large numbers", 9999, 100, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getTotalPagesFunc(tc.totalItems, tc.itemsPerPage)
			if result != tc.expected {
				t.Errorf("For %d items and %d per page, expected %d pages, got %d", tc.totalItems, tc.itemsPerPage, tc.expected, result)
			}
		})
	}
}

func TestHelperFunctionsExistInMap(t *testing.T) {
	helperCtx := HelperContext{Tz: time.UTC}
	funcMap := HelperFunctionsMapping(helperCtx)

	expectedFunctions := []string{
		"formatTime",
		"totalGreater",
		"isNotFirst",
		"isNotLast",
		"trimSample",
		"sub",
		"replaceWebp",
		"buildTime",
		"getTotalPages",
	}

	for _, funcName := range expectedFunctions {
		if _, ok := funcMap[funcName]; !ok {
			t.Errorf("Expected function '%s' to be in FuncMap, but it was not found", funcName)
		}
	}
}

// Example of testing a template that uses these helpers
func TestTemplateWithHelpers(t *testing.T) {
	helperCtx := HelperContext{Tz: time.UTC}
	funcMap := HelperFunctionsMapping(helperCtx)

	tmplSrc := `
Date: {{ .Time | formatTime }}
Total Greater: {{ totalGreater .Total .Limit }}
Not First: {{ isNotFirst .Offset }}
Not Last: {{ isNotLast .Limit .Offset .Total }}
Trimmed: {{ .Body | trimSample }}
Subtracted: {{ sub .A .B }}
WebP: {{ .ImgHTML | replaceWebp }}
Build Time: {{ buildTime }}
Total Pages: {{ getTotalPages .TotalItems .ItemsPerPage }}
`
	tmpl, err := template.New("test").Funcs(funcMap).Parse(tmplSrc)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := struct {
		Time         time.Time
		Total        int
		Limit        int
		Offset       int
		Body         string
		A            int
		B            int
		ImgHTML      string
		TotalItems   int
		ItemsPerPage int
	}{
		Time:         time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC),
		Total:        100,
		Limit:        10,
		Offset:       0,
		Body:         "<p>Hello World</p> This is a test.",
		A:            10,
		B:            3,
		ImgHTML:      `<img src="https://images.microcms-assets.io/assets/image.png">`,
		TotalItems:   55,
		ItemsPerPage: 10,
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	// Basic checks for output, more specific checks can be added
	if !strings.Contains(result.String(), "Date: 2024-01-15") {
		t.Errorf("formatTime helper did not produce expected output. Got: %s", result.String())
	}
	if !strings.Contains(result.String(), "Trimmed: Hello World This is a test.…") {
		t.Errorf("trimSample helper did not produce expected output. Got: %s", result.String())
	}
	if !strings.Contains(result.String(), "WebP: <img src=\"https://images.microcms-assets.io/assets/image.png?fm=webp\">") {
		t.Errorf("replaceWebp helper did not produce expected output. Got: %s", result.String())
	}
	if !strings.Contains(result.String(), "Total Pages: 6") {
		t.Errorf("getTotalPages helper did not produce expected output. Got: %s", result.String())
	}
	if !strings.Contains(result.String(), "Build Time: ") { // Check if buildTime produced any output
		t.Errorf("buildTime helper did not produce output. Got: %s", result.String())
	}
}
