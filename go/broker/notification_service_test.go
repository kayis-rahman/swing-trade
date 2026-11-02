package broker

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscordService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer server.Close()
	d := NewDiscordServiceWithClient(server.URL, true, server.Client())
	if !d.SendMessage("hi") {
		t.Fatal("send message false")
	}
	if !d.SendEmbed("title", "desc", ColorGreen) {
		t.Fatal("embed false")
	}
	if d, _ := func() (*DiscordService, bool) { return NewDiscordService("", false), false }(); d.IsEnabled() {
		t.Fatal("disabled should be false")
	}
	disabled := NewDiscordService("url", false)
	if disabled.SendMessage("x") {
		t.Fatal("disabled send should false")
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer bad.Close()
	if NewDiscordServiceWithClient(bad.URL, true, bad.Client()).SendMessage("x") {
		t.Fatal("bad status should false")
	}
}

func TestTelegramService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer server.Close()
	tg := NewTelegramServiceWithClient("token", "chat", true, server.Client(), server.URL)
	if !tg.SendMessage("hi") {
		t.Fatal("telegram send false")
	}
	disabled := NewTelegramService("token", "chat", false)
	if disabled.SendMessage("x") {
		t.Fatal("disabled telegram should false")
	}
	if NewTelegramService("", "chat", true).IsEnabled() {
		t.Fatal("empty token enabled")
	}
}
