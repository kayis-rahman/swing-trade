package domain

import "time"

// NewsArticle mirrors Java's com.swingtrade.domain.NewsArticle record.
// PublishedDate is nullable to match Java's ZonedDateTime field.
type NewsArticle struct {
	Symbol        string
	Title         string
	Link          string
	Description   string
	PublishedDate *time.Time
	Source        string
	RawContent    string
}
