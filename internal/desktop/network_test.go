package desktop

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bitoolkit/biternet-go/client"
)

func TestSessionConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		action       string
		username     string
		changed      bool
		lostResponse bool
		rejected     bool
		cancel       bool
		delay        int32
	}{
		{name: "login waits for session", action: "login", username: "student", changed: true, delay: 1},
		{name: "lost login response", action: "login", username: "student", changed: true, lostResponse: true},
		{name: "wrong password", action: "login", username: "student", rejected: true},
		{name: "different account", action: "login", username: "other", changed: true},
		{name: "logout waits for session", action: "logout", username: "student", changed: true, delay: 1},
		{name: "lost logout response", action: "logout", username: "student", changed: true, lostResponse: true},
		{name: "canceled confirmation", action: "login", username: "student", cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var actions, confirmations atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					http.Redirect(w, r, "/portal?ac_id=1", 302)
				case "/portal":
					fmt.Fprint(w, "portal")

				case "/cgi-bin/rad_user_info":
					online := tc.action == "logout"
					if actions.Load() > 0 && confirmations.Add(1) > tc.delay && tc.changed {
						online = !online
					}

					if online {
						fmt.Fprintf(w, `jsonp({"error":"ok","online_ip":"10.1.2.3","user_name":%q})`, tc.username)
					} else {
						fmt.Fprint(w, `jsonp({"error":"not_online_error","online_ip":"10.1.2.3"})`)
					}

				case "/cgi-bin/get_challenge":
					fmt.Fprint(w, `jsonp({"challenge":"0123456789abcdef0123456789abcdef01234567"})`)
				case "/cgi-bin/srun_portal":
					actions.Add(1)

					if tc.lostResponse {
						w.WriteHeader(502)
						return
					}
					if tc.rejected {
						fmt.Fprint(w, `jsonp({"error":"E2531","error_msg":"账号或密码错误"})`)
						return
					}

					fmt.Fprint(w, `jsonp({"error":"ok"})`)

				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			gateway, err := client.New(client.Config{PortalURL: server.URL, CaptivePortalURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if tc.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}

			state, err := changeSession(ctx, gateway, request{action: tc.action, username: "student", password: "secret"})
			switch {
			case tc.cancel:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("cancellation: %v", err)
				}

			case tc.rejected:
				var rejection *client.GatewayError
				if !errors.As(err, &rejection) {
					t.Fatalf("lost rejection: %v", err)
				}

			case tc.username != "student":
				if !errors.Is(err, client.ErrAlreadyOnline) {
					t.Fatalf("accepted another account: %v", err)
				}

			default:
				if err != nil || state.Online() != (tc.action == "login") {
					t.Fatalf("unconfirmed session: %+v %v", state, err)
				}
			}

			if actions.Load() != 1 {
				t.Fatalf("authentication was repeated: %d", actions.Load())
			}
		})
	}
}
