package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"golang.org/x/mod/semver"
)

const (
	repositoryURL    = "https://github.com/bitoolkit/biternet-go"
	latestReleaseURL = "https://api.github.com/repos/bitoolkit/biternet-go/releases/latest"
)

type release struct {
	Version string
	URL     string
}

func checkForUpdate(ctx context.Context, httpClient *http.Client, current string) (*release, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "BITERNET/"+current)

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("检查更新：HTTP %d", response.StatusCode)
	}

	var data struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&data); err != nil {
		return nil, err
	}

	if semver.Compare(data.Tag, "v"+current) <= 0 {
		return nil, nil
	}

	return &release{
		Version: strings.TrimPrefix(data.Tag, "v"),
		URL:     repositoryURL + "/releases/tag/" + url.PathEscape(data.Tag),
	}, nil
}

func (a *app) checkUpdates() {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	info, err := checkForUpdate(ctx, http.DefaultClient, Version())
	a.win.Update(func() {
		if a.quitting || err != nil {
			return
		}

		a.update = info
		a.redraw()
	})
}

func (a *app) openLink(link string) {
	go func() {
		err := mygo.Shell.OpenExternal(link)
		if err != nil {
			a.win.Update(func() {
				if a.quitting {
					return
				}

				a.message = "无法打开浏览器，请检查默认浏览器设置"
				a.redraw()
			})
		}
	}()
}
