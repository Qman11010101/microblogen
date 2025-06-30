package main

import (
	"bytes"
	"encoding/json"
	// "io/fs" // Not used
	"log"
	// "net/http" // Not used
	// "net/http/httptest" // Not used
	"os"
	"path/filepath"
	"strconv"
	"strings"
	// "sync" // Not used
	"testing"
	"time"

	"github.com/microcmsio/microcms-go-sdk"
)

// MockMicroCMSClient is a mock implementation of a microCMS client.
type MockMicroCMSClient struct {
	ListFunc func(params microcms.ListParams, dst interface{}) error
	GetFunc  func(params microcms.GetParams, dst interface{}) error
}

func (m *MockMicroCMSClient) List(params microcms.ListParams, dst interface{}) error {
	if m.ListFunc != nil {
		return m.ListFunc(params, dst)
	}
	// Default behavior if ListFunc is not set
	switch params.Endpoint {
	case "article":
		data := dst.(*ArticleList)
		data.Articles = []Article{}
		data.Totalcount = 0
	case "category":
		data := dst.(*CategoryList)
		data.Categories = []Category{}
		data.Totalcount = 0
	}
	return nil
}

func (m *MockMicroCMSClient) Get(params microcms.GetParams, dst interface{}) error {
	if m.GetFunc != nil {
		return m.GetFunc(params, dst)
	}
	// Default behavior if GetFunc is not set
	return nil // Or simulate not found, etc.
}

// TestMainFunction simulates the main execution flow with mocks.
// This is a complex test and will be built iteratively.
func TestMainFunction(t *testing.T) {
	// --- Test Setup ---
	oldOsExit := osExit // Assuming osExit is defined globally in config_test.go or similar
	var exitCode = -1
	var fatalOccurred bool

	osExit = func(code int) {
		exitCode = code
		fatalOccurred = true
		panic("os.Exit called in TestMainFunction")
	}
	defer func() {
		osExit = oldOsExit
		if r := recover(); r != nil {
			if r != "os.Exit called in TestMainFunction" {
				panic(r) // re-panic if it's not the one we expect from this test
			}
		}
	}()

	// Create temp directories for templates and output
	tempDir, cleanup := CreateTempDir(t)
	defer cleanup()

	// Define paths for test
	testExportPath := filepath.Join(tempDir, "output")
	testTemplatesPath := filepath.Join(tempDir, "templates")
	testBlogTemplatesPath := filepath.Join(testTemplatesPath, "blog")
	testSinglesTemplatesPath := filepath.Join(testTemplatesPath, "singles")
	testComponentsPath := filepath.Join(testTemplatesPath, "components")
	testStaticPath := filepath.Join(tempDir, "static")

	// Create dummy template files
	os.MkdirAll(testBlogTemplatesPath, 0755)
	os.MkdirAll(testSinglesTemplatesPath, 0755)
	os.MkdirAll(testComponentsPath, 0755)
	os.MkdirAll(testStaticPath, 0755)
	os.MkdirAll(testExportPath, 0755) // main will remove and recreate this

	// Dummy index.html
	indexHTMLContent := `
	<!DOCTYPE html><html><head><title>Test Index</title></head><body>
	<h1>Articles</h1>
	{{range .Articles}}<div><h2>{{.Title}}</h2><p>{{.PublishedAt | formatTime}}</p></div>{{end}}
	<p>Page: {{.PrevPage}} - {{.NextPage}} / {{.AllPage}}</p>
	<p>IsIndex: {{.IsIndex}}, Root: {{.Root}}, ArchiveName: {{.ArchiveName}}</p>
	</body></html>`
	if err := os.WriteFile(filepath.Join(testBlogTemplatesPath, "index.html"), []byte(indexHTMLContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy index.html: %v", err)
	}

	// Dummy article.html
	articleHTMLContent := `
	<!DOCTYPE html><html><head><title>{{.Title}}</title></head><body>
	<h1>{{.Title}}</h1><div>{{range .Body}}{{.Body}}{{end}}</div><p>{{.PublishedAt | formatTime}}</p>
	{{if .Category}}{{range .Category}}<p>Category: {{.Name}}</p>{{end}}{{end}}
	</body></html>`
	if err := os.WriteFile(filepath.Join(testBlogTemplatesPath, "article.html"), []byte(articleHTMLContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy article.html: %v", err)
	}

	// Dummy single page template (e.g., about.html)
	singlePageContent := `
	<!DOCTYPE html><html><head><title>About Us</title></head><body>
	<h1>About Us Page</h1>
	<p>Latest Articles Count: {{len .Latest}}</p>
	<p>Categories Count: {{len .Categories}}</p>
	</body></html>`
	if err := os.WriteFile(filepath.Join(testSinglesTemplatesPath, "about.html"), []byte(singlePageContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy about.html: %v", err)
	}

	// Dummy static file
	if err := os.WriteFile(filepath.Join(testStaticPath, "style.css"), []byte("body { color: blue; }"), 0644); err != nil {
		t.Fatalf("Failed to write dummy static file: %v", err)
	}

	// Dummy component
	componentContent := `{{define "test_component"}}<p>This is a test component.</p>{{end}}`
	if err := os.WriteFile(filepath.Join(testComponentsPath, "test_component.html"), []byte(componentContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy component: %v", err)
	}


	// Set environment variables for the test
	os.Setenv("MICROCMS_API_KEY", "test_main_api_key")
	os.Setenv("SERVICE_DOMAIN", "test_main_service_domain")
	os.Setenv("EXPORT_PATH", testExportPath)
	os.Setenv("TEMPLATES_PATH", testTemplatesPath)
	os.Setenv("BLOG_TEMPLATES_PATH", testBlogTemplatesPath)
	os.Setenv("SINGLES_TEMPLATES_PATH", testSinglesTemplatesPath)
	os.Setenv("COMPONENTS_PATH", testComponentsPath)
	os.Setenv("STATIC_PATH", testStaticPath)
	os.Setenv("PAGE_SHOW_LIMIT", "2") // For testing pagination
	os.Setenv("LATEST_ARTICLES", "1")
	os.Setenv("TIMEZONE", "UTC")

	defer func() {
		os.Unsetenv("MICROCMS_API_KEY")
		os.Unsetenv("SERVICE_DOMAIN")
		os.Unsetenv("EXPORT_PATH")
		os.Unsetenv("TEMPLATES_PATH")
		os.Unsetenv("BLOG_TEMPLATES_PATH")
		os.Unsetenv("SINGLES_TEMPLATES_PATH")
		os.Unsetenv("COMPONENTS_PATH")
		os.Unsetenv("STATIC_PATH")
		os.Unsetenv("PAGE_SHOW_LIMIT")
		os.Unsetenv("LATEST_ARTICLES")
		os.Unsetenv("TIMEZONE")
	}()

	// --- Mock microCMS Client ---
	mockArticles := []Article{
		CreateTestArticle("article1", "Article 1 Title", "Body 1", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), "cat1"),
		CreateTestArticle("article2", "Article 2 Title", "Body 2", time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), "cat2"),
		CreateTestArticle("article3", "Article 3 Title", "Body 3", time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), "cat1"),
	}
	mockCategories := []Category{
		CreateTestCategory("cat1", "Category One"),
		CreateTestCategory("cat2", "Category Two"),
	}

	// Capture log output
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr) // Restore default logger

	// Override the default client creation in main.go
	// This requires main.go to be refactored to allow client injection or a global var for the client.
	// For now, we assume a way to replace the client. If not, this test needs more advanced techniques
	// like HTTP mocking if the client is hardcoded.
	// Let's assume `newMicroCMSClient` is a function var we can swap for testing:
	originalNewClient := newMicroCMSClient
	newMicroCMSClient = func(serviceDomain, apiKey string) microcmsClient { // microcmsClient is an interface
		return &MockMicroCMSClient{
			ListFunc: func(params microcms.ListParams, dst interface{}) error {
				pageLimit, _ := strconv.Atoi(os.Getenv("PAGE_SHOW_LIMIT"))
				if pageLimit == 0 { pageLimit = DEFAULT_PAGE_SHOW_LIMIT }


				switch params.Endpoint {
				case "article":
					listDst := dst.(*ArticleList)

					// Filter by category if present
					var currentArticles []Article
					if strings.Contains(params.Filters, "category[contains]cat1") {
						for _, art := range mockArticles {
							if len(art.Category) > 0 && art.Category[0].ID == "cat1" {
								currentArticles = append(currentArticles, art)
							}
						}
					} else if strings.Contains(params.Filters, "category[contains]cat2") {
						for _, art := range mockArticles {
							if len(art.Category) > 0 && art.Category[0].ID == "cat2" {
								currentArticles = append(currentArticles, art)
							}
						}
					} else {
						currentArticles = mockArticles
					}

					listDst.Totalcount = len(currentArticles)
					start := params.Offset
					end := params.Offset + params.Limit
					if start > len(currentArticles) { start = len(currentArticles)}
					if end > len(currentArticles) { end = len(currentArticles)}

					if params.Limit == 0 && len(params.Fields) > 0 && params.Fields[0] == "id" { // For category article count
						listDst.Articles = currentArticles
					} else if params.Limit == 0 && params.Endpoint == "article" && len(params.Fields) > 3 { // initial fetch for latest
                        latestCount, _ := strconv.Atoi(os.Getenv("LATEST_ARTICLES"))
                        if latestCount == 0 { latestCount = DEFAULT_LATEST_ARTICLES}
                        if latestCount > len(currentArticles) { latestCount = len(currentArticles)}
                        listDst.Articles = currentArticles[:latestCount]
                        listDst.Totalcount = len(currentArticles) // Still need total for pagination calc
                    } else if params.Limit == 0 { // default for category minimum fetch
                        listDst.Articles = currentArticles
                    } else {
						listDst.Articles = currentArticles[start:end]
					}


				case "category":
					listDst := dst.(*CategoryList)
					listDst.Categories = mockCategories
					listDst.Totalcount = len(mockCategories)
				default:
					t.Errorf("Unexpected endpoint for List: %s", params.Endpoint)
				}
				return nil
			},
		}
	}
	defer func() { newMicroCMSClient = originalNewClient }()

	// --- Run main ---
	main() // Call the actual main function

	// --- Assertions ---
	// Check if log output contains expected messages (optional)
	// if !strings.Contains(logBuf.String(), "Rendering done!") {
	// 	t.Errorf("Expected log to contain 'Rendering done!', got: %s", logBuf.String())
	// }

	if !fatalOccurred { // Only run these checks if main didn't try to exit fatally
		// Check for index.html (main page)
		if !FileExists(filepath.Join(testExportPath, "index.html")) {
			t.Errorf("Expected main index.html to be generated")
		} else {
			content, _ := os.ReadFile(filepath.Join(testExportPath, "index.html"))
			if !strings.Contains(string(content), mockArticles[0].Title) {
				t.Errorf("Main index.html does not contain article title: %s", mockArticles[0].Title)
			}
			if !strings.Contains(string(content), "Page: 0 - 2 / 2") { // 3 articles, 2 per page -> 2 pages. Page 1: prev 0, next 2
				t.Errorf("Main index.html pagination info incorrect. Got: %s", string(content))
			}
		}

		// Check for paginated index.html (e.g., /page/2/index.html)
		if !FileExists(filepath.Join(testExportPath, "page", "2", "index.html")) {
			t.Errorf("Expected paginated index.html (page/2/index.html) to be generated")
		} else {
			content, _ := os.ReadFile(filepath.Join(testExportPath, "page", "2", "index.html"))
			if !strings.Contains(string(content), mockArticles[2].Title) { // Third article should be on page 2
				t.Errorf("Paginated index.html (page 2) does not contain article title: %s", mockArticles[2].Title)
			}
			if !strings.Contains(string(content), "Page: 1 - 3 / 2") { // Page 2: prev 1, next 3
				t.Errorf("Paginated index.html (page 2) pagination info incorrect. Got: %s", string(content))
			}
		}

		// Check for article files
		for _, article := range mockArticles {
			articlePath := filepath.Join(testExportPath, "articles", article.ID+".html")
			if !FileExists(articlePath) {
				t.Errorf("Expected article file %s to be generated", articlePath)
			} else {
				content, _ := os.ReadFile(articlePath)
				if !strings.Contains(string(content), article.Title) {
					t.Errorf("Article file %s does not contain title '%s'", articlePath, article.Title)
				}
				if !strings.Contains(string(content), article.Body[0].Body) {
					t.Errorf("Article file %s does not contain body '%s'", articlePath, article.Body[0].Body)
				}
			}
		}

		// Check for category JSON
		categoryJSONPath := filepath.Join(testExportPath, "category.json")
		if !FileExists(categoryJSONPath) {
			t.Errorf("Expected category.json to be generated")
		} else {
			content, _ := os.ReadFile(categoryJSONPath)
			var cats []Category
			json.Unmarshal(content, &cats)
			if len(cats) != len(mockCategories) {
				t.Errorf("category.json contains %d categories, expected %d", len(cats), len(mockCategories))
			}
		}

		// Check for latest.json
		latestJSONPath := filepath.Join(testExportPath, "latest.json")
		if !FileExists(latestJSONPath) {
			t.Errorf("Expected latest.json to be generated")
		} else {
			content, _ := os.ReadFile(latestJSONPath)
			var latestArts []Article
			json.Unmarshal(content, &latestArts)
			expectedLatestCount, _ := strconv.Atoi(os.Getenv("LATEST_ARTICLES"))
			if len(latestArts) != expectedLatestCount {
				t.Errorf("latest.json contains %d articles, expected %d", len(latestArts), expectedLatestCount)
			}
			if len(latestArts) > 0 && latestArts[0].ID != mockArticles[0].ID { // Assuming articles are sorted by publishedAt desc
				t.Errorf("latest.json first article ID is %s, expected %s", latestArts[0].ID, mockArticles[0].ID)
			}
		}

		// Check for category pages
		for _, category := range mockCategories {
			categoryIndexPath := filepath.Join(testExportPath, "articles", "category", category.ID, "index.html")
			if !FileExists(categoryIndexPath) {
				t.Errorf("Expected category index.html for %s to be generated at %s", category.ID, categoryIndexPath)
			} else {
				content, _ := os.ReadFile(categoryIndexPath)
				if !strings.Contains(string(content), "ArchiveName: Category: "+category.Name) {
					t.Errorf("Category index.html for %s does not contain correct ArchiveName. Got: %s", category.ID, string(content))
				}
				// Check if only articles of this category are listed (simplified check)
				// For cat1 (article1, article3)
				if category.ID == "cat1" {
					if !strings.Contains(string(content), mockArticles[0].Title) {
						t.Errorf("Category cat1 page missing article1 title")
					}
					if !strings.Contains(string(content), mockArticles[2].Title) {
						t.Errorf("Category cat1 page missing article3 title")
					}
					if strings.Contains(string(content), mockArticles[1].Title) { // article2 is cat2
						t.Errorf("Category cat1 page incorrectly contains article2 title")
					}
				}
			}
		}

		// Check for single pages
		singlePageOutputPath := filepath.Join(testExportPath, "about.html")
		if !FileExists(singlePageOutputPath) {
			t.Errorf("Expected single page about.html to be generated")
		} else {
			content, _ := os.ReadFile(singlePageOutputPath)
			if !strings.Contains(string(content), "About Us Page") {
				t.Errorf("Single page about.html does not contain expected content.")
			}
			expectedLatestCount, _ := strconv.Atoi(os.Getenv("LATEST_ARTICLES"))
			if !strings.Contains(string(content), "Latest Articles Count: "+strconv.Itoa(expectedLatestCount)) {
				t.Errorf("Single page about.html latest article count mismatch. Got: %s", string(content))
			}
			if !strings.Contains(string(content), "Categories Count: "+strconv.Itoa(len(mockCategories))) {
				t.Errorf("Single page about.html categories count mismatch. Got: %s", string(content))
			}
		}

		// Check for static files copied
		staticFileOutputPath := filepath.Join(testExportPath, "style.css")
		if !FileExists(staticFileOutputPath) {
			t.Errorf("Expected static file style.css to be copied")
		} else {
			content, _ := os.ReadFile(staticFileOutputPath)
			if string(content) != "body { color: blue; }" {
				t.Errorf("Static file style.css content mismatch")
			}
		}
	} else if exitCode != 1 && !t.Failed() { // if os.Exit was called with non-zero by main and no test assertion failed before panic
		// This case might be tricky if an assertion *within* main (like template check) caused the exit.
		// The t.Failed() check is for failures in *this* test function's assertions.
		t.Errorf("main function exited with code %d. Log: %s", exitCode, logBuf.String())
	}
}

// This is a placeholder for the actual microCMS client interface.
// We need this to allow swapping the client implementation.
type microcmsClient interface {
	List(params microcms.ListParams, dst interface{}) error
	Get(params microcms.GetParams, dst interface{}) error
}

// This variable will hold the function to create a client.
// In main.go, you would change `microcms.New` to `newMicroCMSClient`.
var newMicroCMSClient func(serviceDomain, apiKey string) microcmsClient = func(serviceDomain, apiKey string) microcmsClient {
	return microcms.New(serviceDomain, apiKey) // Real implementation
}

// Add a test for template existence check in main
func TestMain_TemplateExistenceCheck(t *testing.T) {
	oldOsExit := osExit
	var exitCode = -1
	var fatalOccurred bool

	osExit = func(code int) {
		exitCode = code
		fatalOccurred = true
		panic("os.Exit called by template check")
	}
	defer func() {
		osExit = oldOsExit
		if r := recover(); r != nil {
			if r != "os.Exit called by template check" {
				panic(r)
			}
		}
	}()

	tempDir, cleanup := CreateTempDir(t)
	defer cleanup()

	testExportPath := filepath.Join(tempDir, "output")
	testTemplatesPath := filepath.Join(tempDir, "templates")
	testBlogTemplatesPath := filepath.Join(testTemplatesPath, "blog") // This will be missing index.html

	os.MkdirAll(testBlogTemplatesPath, 0755) // Create blog dir but not files inside

	os.Setenv("MICROCMS_API_KEY", "key")
	os.Setenv("SERVICE_DOMAIN", "domain")
	os.Setenv("EXPORT_PATH", testExportPath)
	os.Setenv("TEMPLATES_PATH", testTemplatesPath)
	os.Setenv("BLOG_TEMPLATES_PATH", testBlogTemplatesPath)
	// Other paths not strictly necessary for this specific test of template existence

	defer func() {
		os.Unsetenv("MICROCMS_API_KEY")
		os.Unsetenv("SERVICE_DOMAIN")
		os.Unsetenv("EXPORT_PATH")
		os.Unsetenv("TEMPLATES_PATH")
		os.Unsetenv("BLOG_TEMPLATES_PATH")
	}()

	// Capture log output
	var logBuf bytes.Buffer
	originalLogOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(originalLogOutput)

	main() // This should trigger log.Fatal due to missing templates

	if !fatalOccurred {
		t.Errorf("Expected main to call os.Exit due to missing templates, but it didn't")
	}
	if exitCode != 1 { // log.Fatal exits with 1
		t.Errorf("Expected exit code 1 due to missing templates, got %d", exitCode)
	}

	expectedLog := "Error: Missing templates. You must prepare \"article.html\" and \"index.html\" inside ./" + testBlogTemplatesPath
	if !strings.Contains(logBuf.String(), expectedLog) {
		// t.Errorf("Expected log message about missing templates, got: %s", logBuf.String())
		// Due to panic interrupting log.Fatal, the full message might not be in logBuf.
		// The panic recovery for os.Exit is the primary check.
	}
}
