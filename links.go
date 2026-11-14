package main

// Links that aren't a file to download (Party > Add Link…, App.AddURL):
//
//   - a live stream (rtmp://, rtsp://, srt://, or an http(s) .m3u8 or .mpd playlist): a row
//     that plays straight from the link, for as long as the stream runs. Nothing is
//     downloaded, there's nothing to seek and no place kept: Item.Live, its Path the link.
//   - a video site's page (YouTube, Vimeo and the rest of what yt-dlp knows): yt-dlp,
//     bundled beside ffmpeg, finds the video behind it. A live one is played as a live
//     stream; any other is downloaded as a download link is, under the video's title,
//     and plays as it comes down.

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"os/exec"
	"path"
	"strings"
	"time"
)

// liveLink: a link ffmpeg plays as a stream, not a file to fetch
func liveLink(u *url.URL) bool {
	switch strings.ToLower(u.Scheme) {
	case "rtmp", "rtmps", "rtsp", "rtsps", "srt":
		return u.Host != ""
	case "http", "https":
		e := strings.ToLower(path.Ext(u.Path))
		return u.Host != "" && (e == ".m3u8" || e == ".mpd")
	}
	return false
}

// streamTitle: a live row's name, from its link
func streamTitle(u *url.URL) string {
	name := strings.TrimSuffix(path.Base(u.Path), path.Ext(u.Path))
	if name == "" || name == "." || name == "/" || name == "index" || name == "playlist" || name == "master" || name == "live" {
		return "Live: " + u.Hostname()
	}
	return "Live: " + name
}

// siteVideo is what yt-dlp found behind a page
type siteVideo struct {
	URL   string // a live stream's playlist, or the video's own file
	Title string
	Live  bool
}

// ytDlp: the bundled yt-dlp, or one on the PATH; "" if there's none
func ytDlp() string { return findTool("yt-dlp") }

// findSiteVideo asks yt-dlp what's behind a page: a single file with sound and
// picture in it (so it downloads, and plays, as any other), or a live stream's playlist
func findSiteVideo(page string) (*siteVideo, error) {
	bin := ytDlp()
	if bin == "" {
		return nil, errors.New("no yt-dlp")
	}
	ctx, cancel := contextTimeout(90 * time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--no-playlist", "--no-warnings", "--dump-single-json",
		"-f", "best[acodec!=none][vcodec!=none][ext=mp4]/best[acodec!=none][vcodec!=none]/best", page)
	hidden(cmd)
	var out, said bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &said
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(said.String())
		if i := strings.LastIndex(msg, "ERROR: "); i >= 0 {
			msg = msg[i+len("ERROR: "):]
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	var info struct {
		Title  string `json:"title"`
		URL    string `json:"url"`
		IsLive bool   `json:"is_live"`
		Proto  string `json:"protocol"`
	}
	if err := json.Unmarshal(out.Bytes(), &info); err != nil || info.URL == "" {
		return nil, errors.New("yt-dlp found no single video there")
	}
	live := info.IsLive || strings.Contains(info.Proto, "m3u8") || strings.Contains(info.Proto, "dash")
	return &siteVideo{URL: info.URL, Title: strings.TrimSpace(info.Title), Live: live}, nil
}

// unsupported: yt-dlp doesn't know the site (so the link is tried as a plain download)
func unsupported(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Unsupported URL") || err.Error() == "no yt-dlp")
}

// addLive puts a live stream's row in the list, and hosts
func (a *App) addLive(link, title string) string {
	result := ""
	a.wait(func() {
		if a.locked {
			result = "The party is sealed: open it first"
			return
		}
		for _, it := range a.cfg.Queue {
			if it.Live && it.Path == link {
				result = "That stream is in the list already"
				return
			}
		}
		it := &Item{Path: link, Kind: "video", Live: true, Title: title}
		a.editQueue(func(q []*Item) []*Item { return append(q, it) }, nil)
		a.log("Added the live stream " + title)
		if a.host == nil || !a.host.Running() {
			a.hostStart()
		}
	})
	return result
}

// notArchive: not an Internet Archive page (archive.go reads those itself)
func notArchive(u *url.URL) bool {
	item, _ := archiveItem(u)
	return item == ""
}
