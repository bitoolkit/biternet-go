package desktop

import (
	"context"
	"errors"
	"time"

	"github.com/bitoolkit/biternet-go/client"
)

// The desktop controls its own connection. Raw client operations also support
// other IPs, whose session cannot be confirmed through the local status endpoint.
func changeSession(ctx context.Context, gateway *client.Client, req request) (client.State, error) {
	actionCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	var actionErr error
	if req.action == "login" {
		_, actionErr = gateway.Login(actionCtx, req.username, req.password)
	} else {
		_, actionErr = gateway.Logout(actionCtx, req.username)
	}
	cancel()

	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(500 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return client.State{}, ctx.Err()
			case <-timer.C:
			}
		}

		state, err := gateway.Status(ctx)

		if err == nil {
			if req.action == "logout" && !state.Online() {
				return state, nil
			}
			if req.action == "login" && state.Online() {
				if state.Username == req.username {
					return state, nil
				}

				return client.State{}, client.ErrAlreadyOnline
			}
		}

		// A lost response may still have changed the session. One query recovers
		// that success; an actual rejection never resends credentials here.
		if actionErr != nil {
			return client.State{}, actionErr
		}
		if ctx.Err() != nil {
			return client.State{}, ctx.Err()
		}
		lastErr = err
	}

	if lastErr != nil {
		return client.State{}, lastErr
	}

	return client.State{}, errors.New("网关尚未确认连接状态，请稍后重试")
}
