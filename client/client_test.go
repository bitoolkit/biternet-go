package client_test

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bitoolkit/biternet-go/client"
)

func TestLoginAndLogout(t *testing.T) {
	online := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		switch r.URL.Path {
		case "/", "/captive":
			http.Redirect(w, r, "/portal?ac_id=1", http.StatusFound)
		case "/portal":
			fmt.Fprint(w, "portal")
		case "/cgi-bin/rad_user_info":
			if online {
				fmt.Fprint(w, `jsonp({"error":"ok","online_ip":"10.1.2.3","user_name":"20230001","sum_bytes":1024,"sum_seconds":60,"user_balance":48.5})`)
			} else {
				fmt.Fprint(w, `jsonp({"error":"not_online_error","online_ip":"10.1.2.3"})`)
			}
		case "/cgi-bin/get_challenge":
			if q.Get("username") != "20230001" || q.Get("ip") != "10.1.2.3" {
				t.Error("incorrect challenge identity")
			}
			fmt.Fprint(w, `jsonp({"challenge":"0123456789abcdef0123456789abcdef01234567"})`)
		case "/cgi-bin/srun_portal":
			if q.Get("callback") != "jsonp" || q.Get("ac_id") != "1" || q.Get("ip") != "10.1.2.3" {
				t.Errorf("incorrect authentication parameters")
			}
			if q.Get("action") == "login" {
				// Vectors from the existing Rust client, checked at the public HTTP boundary.
				want := map[string]string{
					"password": "{MD5}24d9b760ae70cb5b50264a78372a1004",
					"info":     "{SRBX1}aaBpnCRhLjgRzvCU9TQW2rl6GYq6VWEyIlh63hNUHudeLeN4TEyJcisCb7AyRlwCRhHjDkpoybaAKZ058Ze054hmKxJ/mdXrhtopglEPJdppFY1yC8qOV9PnmJwp8aaq",
					"chksum":   "7be3d02ba4c337f752b18e6ba4f7b7c97b721427",
					"n":        "200",
					"type":     "1",
				}
				for key, value := range want {
					if q.Get(key) != value {
						t.Errorf("%s does not match Rust vector: %s", key, q.Get(key))
					}
				}
				online = true
			} else {
				if q.Get("action") != "logout" || q.Get("username") != "20230001" {
					t.Error("incorrect logout identity")
				}
				if q.Has("password") || q.Has("info") {
					t.Error("logout transmitted credentials")
				}
				online = false
			}
			fmt.Fprint(w, `jsonp({"error":"ok"})`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, err := client.New(client.Config{PortalURL: server.URL, CaptivePortalURL: server.URL + "/captive"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := c.Login(context.Background(), "20230001", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if response.Error != "ok" {
		t.Fatalf("unexpected response: %+v", response)
	}

	state, err := c.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !state.Online() || state.UsedBytes == nil || *state.UsedBytes != 1024 || state.Balance == nil || *state.Balance != 48.5 {
		t.Fatalf("unexpected state: %+v", state)
	}

	if _, err := c.Login(context.Background(), "other", "secret"); !errors.Is(err, client.ErrAlreadyOnline) {
		t.Fatalf("other account: %v", err)
	}

	if _, err := c.Logout(context.Background(), "fallback"); err != nil {
		t.Fatal(err)
	}
	state, err = c.Status(context.Background())
	if err != nil || state.Online() {
		t.Fatalf("logout not reflected: %+v %v", state, err)
	}
	if _, err := c.Logout(context.Background(), "fallback"); err != nil {
		t.Fatal(err)
	}
}

func TestStatusFailuresAndMissingCounters(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		valid      bool
		online     bool
	}{
		{"offline", `jsonp({"error":"not_online_error","online_ip":"10.1.2.3"});` + "\n", 200, true, false},
		{"offline with username", `jsonp({"error":"not_online_error","online_ip":"10.1.2.3","user_name":"student"})`, 200, true, false},
		{"online without counters", `jsonp({"error":"ok","online_ip":"10.1.2.3","user_name":"student"})`, 200, true, true},
		{"online without error field", `jsonp({"online_ip":"10.1.2.3","user_name":"student"})`, 200, true, true},
		{"unrecognized gateway result", `jsonp({"error":"server_error","online_ip":"10.1.2.3"})`, 200, false, false},
		{"error with session fields", `jsonp({"error":"server_error","online_ip":"10.1.2.3","user_name":"student","sum_seconds":60})`, 200, false, false},
		{"invalid IP", `jsonp({"error":"ok","online_ip":"not an IP"})`, 200, false, false},
		{"wrong callback", `callback({"error":"ok"})`, 200, false, false},
		{"malformed JSON", `jsonp({secret})`, 200, false, false},
		{"HTTP failure", `jsonp({"error":"ok","online_ip":"10.1.2.3"})`, 503, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer s.Close()

			c, err := client.New(client.Config{PortalURL: s.URL})
			if err != nil {
				t.Fatal(err)
			}
			state, err := c.Status(context.Background())
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, err=%v", tc.valid, err)
			}
			if tc.valid && state.Online() != tc.online {
				t.Fatalf("online=%v, want %v", state.Online(), tc.online)
			}
			if tc.valid && (state.UsedBytes != nil || state.UsedSeconds != nil || state.Balance != nil) {
				t.Fatal("missing counters replaced by fabricated data")
			}
			if err != nil && strings.Contains(err.Error(), "secret") {
				t.Fatal("raw gateway body leaked")
			}
		})
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, &url.Error{Op: "Get", URL: r.URL.String(), Err: context.DeadlineExceeded}
}

func TestTransportErrorPreservesCauseWithoutURL(t *testing.T) {
	c, err := client.New(client.Config{HTTPClient: &http.Client{Transport: failingTransport{}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Status(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lost cause: %v", err)
	}
	if strings.Contains(err.Error(), "http://") {
		t.Fatalf("request URL leaked: %v", err)
	}
}

func TestAuthenticationAcknowledgementAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		body     string
		rejected bool
	}{
		{name: "explicit success", body: `{"error":"ok"}`},
		{name: "success without error field", body: `{"suc_msg":"login_ok"}`},
		{name: "error with success field", body: `{"error":"E2531","error_msg":"账号或密码错误","suc_msg":""}`, rejected: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					http.Redirect(w, r, "/portal?ac_id=1", 302)
				case "/portal":
					fmt.Fprint(w, "portal")
				case "/cgi-bin/rad_user_info":
					fmt.Fprint(w, `jsonp({"error":"not_online_error","online_ip":"10.1.2.3"})`)
				case "/cgi-bin/get_challenge":
					fmt.Fprint(w, `jsonp({"challenge":"0123456789abcdef0123456789abcdef01234567"})`)
				case "/cgi-bin/srun_portal":
					fmt.Fprintf(w, "jsonp(%s)", tc.body)
				}
			}))
			defer s.Close()

			c, err := client.New(client.Config{PortalURL: s.URL, CaptivePortalURL: s.URL})
			if err != nil {
				t.Fatal(err)
			}
			response, err := c.Login(context.Background(), "student", "secret")
			if tc.rejected {
				var rejection *client.GatewayError
				if !errors.As(err, &rejection) || rejection.Code != "E2531" {
					t.Fatalf("lost rejection: %v", err)
				}
			} else if err != nil {
				t.Fatalf("acknowledgement: %+v %v", response, err)
			}

			state, err := c.Status(context.Background())
			if err != nil || state.Online() {
				t.Fatalf("session unexpectedly online: %+v %v", state, err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err = c.Login(ctx, "student", "secret")
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
		})
	}
}

func TestForceAndDumbTerminalOperations(t *testing.T) {
	for _, dumb := range []bool{false, true} {
		t.Run(fmt.Sprintf("dumb=%v", dumb), func(t *testing.T) {
			online := true
			loginRequests, logoutRequests := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				switch r.URL.Path {
				case "/":
					http.Redirect(w, r, "/portal?ac_id=7", 302)
				case "/portal":
					fmt.Fprint(w, "portal")
				case "/cgi-bin/rad_user_info":
					if online {
						fmt.Fprint(w, `jsonp({"error":"ok","online_ip":"10.1.2.3","user_name":"existing"})`)
					} else {
						fmt.Fprint(w, `jsonp({"error":"not_online_error","online_ip":"10.1.2.3"})`)
					}
				case "/cgi-bin/get_challenge":
					if q.Get("ip") != "10.9.8.7" {
						t.Error("missing configured IP in challenge")
					}
					fmt.Fprint(w, `jsonp({"challenge":"0123456789abcdef0123456789abcdef01234567"})`)
				case "/cgi-bin/srun_portal", "/cgi-bin/rad_user_dm":
					if q.Get("ip") != "10.9.8.7" {
						t.Error("missing configured authentication IP")
					}
					if q.Get("action") == "login" {
						loginRequests++
					} else {
						logoutRequests++
						wantUser := "fallback"
						if online {
							wantUser = "existing"
						}
						if q.Get("username") != wantUser {
							t.Error("wrong logout account")
						}
						if dumb {
							if r.URL.Path != "/cgi-bin/rad_user_dm" || q.Get("unbind") != "1" || q.Has("ac_id") || q.Has("action") {
								t.Error("wrong dumb terminal request")
							}
							timestamp, err := strconv.ParseInt(q.Get("time"), 10, 64)
							if err != nil || time.Since(time.Unix(timestamp, 0)) > 5*time.Second {
								t.Error("incorrect logout timestamp")
							}
							digest := sha1.Sum([]byte(q.Get("time") + wantUser + "10.9.8.7" + "1" + q.Get("time")))
							if q.Get("sign") != hex.EncodeToString(digest[:]) {
								t.Error("incorrect logout signature")
							}
						} else if q.Get("action") != "logout" || q.Get("ac_id") != "7" {
							t.Error("wrong standard logout request")
						}
					}
					fmt.Fprint(w, `jsonp({"error":"ok","access_token":"token","res":"ok","online_ip":"10.9.8.7"})`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			c, err := client.New(client.Config{
				PortalURL:        server.URL,
				CaptivePortalURL: server.URL,
				IP:               netip.MustParseAddr("10.9.8.7"),
				DumbTerminal:     dumb,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Login(context.Background(), "student", "password"); !errors.Is(err, client.ErrAlreadyOnline) {
				t.Fatalf("normal login: %v", err)
			}

			result, err := c.ForceLogin(context.Background(), "student", "password")
			if err != nil || result.AccessToken == nil || *result.AccessToken != "token" || result.IP != "10.9.8.7" {
				t.Fatalf("force login: %+v %v", result, err)
			}

			if _, err := c.Logout(context.Background(), "fallback"); err != nil {
				t.Fatal(err)
			}

			online = false
			if _, err := c.Logout(context.Background(), "fallback"); err != nil {
				t.Fatal(err)
			}
			if _, err := c.ForceLogout(context.Background(), "fallback"); err != nil {
				t.Fatal(err)
			}

			if loginRequests != 1 || logoutRequests != 2 {
				t.Fatalf("unexpected requests: login=%d logout=%d", loginRequests, logoutRequests)
			}
		})
	}
}

func TestExtendedStatusAndChallengeRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/portal?ac_id=1", 302)
		case "/portal":
			fmt.Fprint(w, "portal")
		case "/cgi-bin/rad_user_info":
			fmt.Fprint(w, `jsonp({"error":"not_online_error","online_ip":"10.1.2.3","remain_bytes":-100,"wallet_balance":0,"real_name":"同学","ServerFlag":1,"client_ip":"10.1.2.3"})`)
		case "/cgi-bin/get_challenge":
			fmt.Fprint(w, `jsonp({"error":"E2531","error_msg":"账号或密码错误","challenge":"0123456789abcdef0123456789abcdef01234567"})`)
		default:
			t.Error("authentication must stop when challenge is rejected")
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, err := client.New(client.Config{PortalURL: server.URL, CaptivePortalURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	state, err := c.Status(context.Background())
	if err != nil ||
		state.RemainBytes == nil || *state.RemainBytes != -100 ||
		state.WalletBalance == nil || *state.WalletBalance != 0 ||
		state.RealName == nil || *state.RealName != "同学" || state.BytesIn != nil {
		t.Fatalf("status: %+v %v", state, err)
	}

	_, err = c.Login(context.Background(), "student", "password")
	var rejection *client.GatewayError
	if !errors.As(err, &rejection) || rejection.Code != "E2531" {
		t.Fatalf("lost rejection: %v", err)
	}
}
