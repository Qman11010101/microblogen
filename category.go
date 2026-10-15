package main

// categoriesWithArticles returns the categories that should appear in the tag
// cloud. The cloud is rendered from this list (category.json), so a category
// whose article count is 0 is omitted. Categories with a count of at least 1
// are kept in their original order. A category with no recorded count is kept.
func categoriesWithArticles(categories []Category, articleCounts map[string]int) []Category {
	if len(categories) == 0 || len(articleCounts) == 0 {
		return categories
	}

	filtered := make([]Category, 0, len(categories))
	removed := false
	for _, cat := range categories {
		count, ok := articleCounts[cat.ID]
		if ok && count <= 0 {
			removed = true
			continue
		}
		filtered = append(filtered, cat)
	}
	if !removed {
		return categories
	}
	return filtered
}
