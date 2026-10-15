package main

import "testing"

func TestCategoriesWithArticlesOmitsZeroCounts(t *testing.T) {
	categories := []Category{
		{ID: "used", Name: "Used"},
		{ID: "empty", Name: "Empty"},
		{ID: "one", Name: "One"},
		{ID: "also-empty", Name: "Also empty"},
	}
	counts := map[string]int{
		"used":       3,
		"empty":      0,
		"one":        1,
		"also-empty": 0,
	}

	got := categoriesWithArticles(categories, counts)
	want := []string{"used", "one"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), idsOf(got))
	}
	for i, id := range want {
		if got[i].ID != id || got[i].Name != categories[indexByID(categories, id)].Name {
			t.Fatalf("got[%d] = %+v, want id %s with its original name", i, got[i], id)
		}
	}
}

func TestCategoriesWithArticlesKeepsPositiveCounts(t *testing.T) {
	categories := []Category{
		{ID: "a", Name: "A"},
		{ID: "b", Name: "B"},
	}
	counts := map[string]int{"a": 1, "b": 8}

	got := categoriesWithArticles(categories, counts)
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("positive counts changed the list: %+v", got)
	}
	if &got[0] != &categories[0] {
		t.Fatal("list with no zero-count tags should be returned unchanged")
	}
}

func TestCategoriesWithArticlesKeepsUnknownCounts(t *testing.T) {
	categories := []Category{{ID: "unrecorded", Name: "Unrecorded"}}
	got := categoriesWithArticles(categories, map[string]int{"other": 0})
	if len(got) != 1 || got[0].ID != "unrecorded" {
		t.Fatalf("category without a recorded count was dropped: %+v", got)
	}
}

func idsOf(categories []Category) []string {
	ids := make([]string, len(categories))
	for i, cat := range categories {
		ids[i] = cat.ID
	}
	return ids
}

func indexByID(categories []Category, id string) int {
	for i, cat := range categories {
		if cat.ID == id {
			return i
		}
	}
	return -1
}
