package broker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// NotificationService is part of the broker notification port.
type NotificationService interface {
	SendMessage(content string) bool
}

// HTTPClient is part of the broker notification port.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// newHTTPClient is part of the broker notification port.
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// DiscordConfig is part of the broker notification port.
type DiscordConfig struct {
	WebhookURL string
	Enabled    bool
}

// DiscordService is part of the broker notification port.
type DiscordService struct {
	webhookURL string
	enabled    bool
	client     HTTPClient
	timeout    time.Duration
}

// NewDiscordService is part of the broker notification port.
func NewDiscordService(webhookURL string, enabled bool) *DiscordService {
	return &DiscordService{webhookURL: webhookURL, enabled: enabled, client: newHTTPClient(10 * time.Second), timeout: 10 * time.Second}
}

// NewDiscordServiceWithClient is part of the broker notification port.
func NewDiscordServiceWithClient(webhookURL string, enabled bool, client HTTPClient) *DiscordService {
	return &DiscordService{webhookURL: webhookURL, enabled: enabled, client: client, timeout: 10 * time.Second}
}

// IsEnabled is part of the broker notification port.
func (d *DiscordService) IsEnabled() bool { return d.enabled && d.webhookURL != "" }

// SendMessage is part of the broker notification port.
func (d *DiscordService) SendMessage(content string) bool {
	if !d.IsEnabled() {
		return false
	}
	return d.post(map[string]any{"content": content})
}

// SendEmbed is part of the broker notification port.
func (d *DiscordService) SendEmbed(title, description string, color int) bool {
	if !d.IsEnabled() {
		return false
	}
	return d.post(map[string]any{"embeds": []map[string]any{{"title": title, "description": description, "color": color}}})
}

// SendEmbedFields is part of the broker notification port.
func (d *DiscordService) SendEmbedFields(title, description string, color int, fields []EmbedField) bool {
	if !d.IsEnabled() {
		return false
	}
	embed := map[string]any{"title": title, "description": description, "color": color}
	if len(fields) > 0 {
		fm := make([]map[string]any, 0, len(fields))
		for _, f := range fields {
			fm = append(fm, f.toMap())
		}
		embed["fields"] = fm
	}
	return d.post(map[string]any{"embeds": []map[string]any{embed}})
}

// post is part of the broker notification port.
func (d *DiscordService) post(payload map[string]any) bool {
	body, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	req, err := http.NewRequest(http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// EmbedField is part of the broker notification port.
type EmbedField struct {
	Name   string
	Value  string
	Inline bool
}

// toMap is part of the broker notification port.
func (f EmbedField) toMap() map[string]any {
	return map[string]any{"name": f.Name, "value": f.Value, "inline": f.Inline}
}

// Constants is part of the broker notification port.
const (
	ColorGreen   = 11625876
	ColorYellow  = 1554693
	ColorRed     = 15158332
	ColorBlurple = 0x5865F2
)
