package main

import (
	"net/url"
	"testing"
)

func TestLiveLinks(t *testing.T) {
	for link, want := range map[string]bool{
		"https://example.com/live/stream.m3u8": true,
		"http://example.com/a/manifest.MPD":    true,
		"rtmp://example.com/live/key":          true,
		"rtsp://192.168.1.20:554/cam":          true,
		"srt://example.com:9000":               true,
		"https://example.com/movie.mp4":        false,
		"https://www.youtube.com/watch?v=abc":  false,
		"ftp://example.com/x.m3u8":             false,
	} {
		u, _ := url.Parse(link)
		if liveLink(u) != want {
			t.Errorf("%s: live is %v", link, !want)
		}
	}
	for link, want := range map[string]string{
		"https://example.com/news/channel7.m3u8": "Live: channel7",
		"https://cdn.example.com/hls/index.m3u8": "Live: cdn.example.com",
	} {
		u, _ := url.Parse(link)
		if got := streamTitle(u); got != want {
			t.Errorf("%s: %q", link, got)
		}
	}
	it := &Item{Path: "https://example.com/x.m3u8", Live: true, Title: "Live: x"}
	if it.title() != "Live: x" {
		t.Fatal(it.title())
	}
}
