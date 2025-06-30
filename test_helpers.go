package main

import (
	"os"
	"testing"
	"time"
)

// CreateTestArticle はテスト用の記事データを作成します。
func CreateTestArticle(id, title, bodyContent string, publishedAt time.Time, categoryName string) Article {
	return Article{
		ID:    id,
		Title: title,
		Body: []Body{
			{Fieldid: "body", Body: bodyContent},
		},
		PublishedAt: publishedAt,
		Category: []Category{
			{ID: categoryName, Name: categoryName},
		},
	}
}

// CreateTestCategory はテスト用のカテゴリデータを作成します。
func CreateTestCategory(id, name string) Category {
	return Category{
		ID:   id,
		Name: name,
	}
}

// CreateTempDir は一時的なテストディレクトリを作成し、そのパスを返します。
// テスト終了時にクリーンアップ関数を呼び出す必要があります。
func CreateTempDir(t *testing.T) (string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "testdir-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	cleanup := func() {
		os.RemoveAll(tempDir)
	}
	return tempDir, cleanup
}

// CreateTempFile は一時的なテストファイルを作成し、そのパスを返します。
func CreateTempFile(t *testing.T, dir, pattern string) *os.File {
	t.Helper()
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return tmpFile
}
