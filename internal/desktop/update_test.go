package desktop

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type releaseTransport func(*http.Request) (*http.Response, error)

func (f releaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReleaseUpdates(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tag     string
		current string
		link    string
		status  int
		update  bool
		fail    bool
	}{
		{name: "current", tag: "v1.2.3", current: "1.2.3"},
		{name: "new patch", tag: "v1.2.4", current: "1.2.3", update: true},
		{name: "numeric ordering", tag: "v1.10.0", current: "1.9.0", update: true},
		{name: "client release", tag: "client/v2.0.0", current: "1.2.3"},
		{name: "old repository tag", tag: "biternet-v2.0.0", current: "1.2.3"},
		{name: "plain", tag: "2.0.0", current: "1.2.3"},
		{name: "prerelease before stable", tag: "v2.0.0-rc.1", current: "2.0.0"},
		{name: "stable after prerelease", tag: "v2.0.0", current: "2.0.0-rc.1", update: true},
		{name: "build metadata", tag: "v1.2.3+new", current: "1.2.3+old"},
		{name: "invalid tag", tag: "latest", current: "1.2.3"},
		{name: "short tag", tag: "v2.0", current: "1.2.3", update: true},
		{name: "no releases", status: 404, current: "1.2.3"},
		{name: "rate limited", status: 403, current: "1.2.3", fail: true},
		{name: "response URL ignored", tag: "v2.0.0", current: "1.2.3", link: "file:///tmp/install", update: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			link := tc.link
			if link == "" {
				link = repositoryURL + "/releases/tag/" + tc.tag
			}

			hc := &http.Client{Transport: releaseTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != latestReleaseURL ||
					req.Header.Get("User-Agent") != "BITERNET/"+tc.current ||
					req.Header.Get("Accept") != "application/vnd.github+json" {
					t.Error("wrong release request")
				}

				status := tc.status
				if status == 0 {
					status = 200
				}
				body := fmt.Sprintf(`{"tag_name":%q,"html_url":%q}`, tc.tag, link)

				return &http.Response{
					StatusCode: status,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			})}

			info, err := checkForUpdate(context.Background(), hc, tc.current)
			if (err != nil) != tc.fail || (info != nil) != tc.update {
				t.Fatalf("update=%+v, error=%v", info, err)
			}
			if info != nil {
				if info.Version != strings.TrimPrefix(tc.tag, "v") {
					t.Fatalf("wrong version: %s", info.Version)
				}
				if info.URL != repositoryURL+"/releases/tag/"+tc.tag {
					t.Fatalf("wrong download URL: %s", info.URL)
				}
			}
		})
	}
}
