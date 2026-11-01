package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPortalURL        = "http://10.0.0.55"
	DefaultCaptivePortalURL = "http://www.bit.edu.cn"
)

// Config's zero value selects the campus gateway and an eight-second HTTP timeout.
type Config struct {
	// PortalURL is the gateway's base URL. Empty selects DefaultPortalURL.
	PortalURL string
	// CaptivePortalURL is used to discover the gateway's access controller ID.
	// Empty selects DefaultCaptivePortalURL.
	CaptivePortalURL string
	// HTTPClient handles gateway requests and redirect discovery. When nil,
	// New creates a client with an eight-second timeout per HTTP request.
	HTTPClient *http.Client
	// IP overrides the address used for login and logout. An invalid address
	// selects the local IP returned by the gateway. IPv6 zones are not supported.
	IP netip.Addr
	// DumbTerminal selects the signed rad_user_dm logout endpoint.
	DumbTerminal bool
}

// Client is safe for concurrent use. Authentication changes for the same network
// connection should be serialized by the caller.
type Client struct {
	portal       *url.URL
	captive      *url.URL
	http         *http.Client
	ip           netip.Addr
	dumbTerminal bool
}

// State contains the gateway's latest session information. Nil counters mean
// that the server did not supply a value, rather than a measured zero.
type State struct {
	Error         string   `json:"error"`
	ErrorMessage  string   `json:"error_msg"`
	IP            string   `json:"online_ip"`
	Username      string   `json:"user_name"`
	UsedBytes     *uint64  `json:"sum_bytes"`
	UsedSeconds   *int64   `json:"sum_seconds"`
	Balance       *float64 `json:"user_balance"`
	ServerFlag    *int64   `json:"ServerFlag,omitempty"`
	AddTime       *int64   `json:"add_time,omitempty"`
	AllBytes      *uint64  `json:"all_bytes,omitempty"`
	BytesIn       *uint64  `json:"bytes_in,omitempty"`
	BytesOut      *uint64  `json:"bytes_out,omitempty"`
	CheckoutDate  *uint64  `json:"checkout_date,omitempty"`
	Domain        *string  `json:"domain,omitempty"`
	GroupID       *string  `json:"group_id,omitempty"`
	KeepaliveTime *uint64  `json:"keepalive_time,omitempty"`
	ProductsName  *string  `json:"products_name,omitempty"`
	RealName      *string  `json:"real_name,omitempty"`
	RemainBytes   *int64   `json:"remain_bytes,omitempty"`
	RemainSeconds *int64   `json:"remain_seconds,omitempty"`
	Sysver        *string  `json:"sysver,omitempty"`
	UserCharge    *int64   `json:"user_charge,omitempty"`
	UserMAC       *string  `json:"user_mac,omitempty"`
	WalletBalance *float64 `json:"wallet_balance,omitempty"`
	ClientIP      *string  `json:"client_ip,omitempty"`
	Result        *string  `json:"res,omitempty"`
	SrunVersion   *string  `json:"srun_ver,omitempty"`
	ST            *int64   `json:"st,omitempty"`
}

// PortalResponse contains the gateway's authentication acknowledgement and any
// access token it returns. A successful response means the request was accepted;
// use [Client.Status] to confirm the resulting session state.
type PortalResponse struct {
	AccessToken  *string `json:"access_token,omitempty"`
	Username     *string `json:"username,omitempty"`
	Success      *string `json:"suc_msg,omitempty"`
	ClientIP     string  `json:"client_ip"`
	IP           string  `json:"online_ip"`
	Error        string  `json:"error"`
	ErrorMessage string  `json:"error_msg"`
	Result       string  `json:"res"`
}

// Online reports whether the gateway's response describes an active session.
func (s State) Online() bool {
	return s.Error == "ok" || (s.Error == "" && (s.Username != "" || s.UsedSeconds != nil))
}

// GatewayError is an authentication or status error returned by the gateway.
type GatewayError struct {
	Code    string
	Message string
}

func (e *GatewayError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Code != "" {
		return e.Code
	}

	return "gateway returned an unknown error"
}

// ErrAlreadyOnline is returned by [Client.Login] when a session is already online.
var ErrAlreadyOnline = errors.New("an account is already online")

// New creates a client without making network requests. It rejects invalid
// gateway URLs and IP addresses with an IPv6 zone.
func New(config Config) (*Client, error) {
	if config.PortalURL == "" {
		config.PortalURL = DefaultPortalURL
	}
	if config.CaptivePortalURL == "" {
		config.CaptivePortalURL = DefaultCaptivePortalURL
	}

	parse := func(raw string) (*url.URL, error) {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" ||
			(u.Scheme != "http" && u.Scheme != "https") ||
			u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("gateway URLs must be HTTP(S) URLs without credentials, queries or fragments")
		}

		return u, nil
	}

	portal, err := parse(config.PortalURL)
	if err != nil {
		return nil, err
	}
	captive, err := parse(config.CaptivePortalURL)
	if err != nil {
		return nil, err
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}

	if config.IP.Zone() != "" {
		return nil, errors.New("gateway IP must not include an IPv6 zone")
	}

	return &Client{
		portal:       portal,
		captive:      captive,
		http:         httpClient,
		ip:           config.IP,
		dumbTerminal: config.DumbTerminal,
	}, nil
}

// Status queries the gateway. An unreachable gateway is an error, not an offline session.
func (c *Client) Status(ctx context.Context) (State, error) {
	var state State
	if err := c.get(ctx, "/cgi-bin/rad_user_info", url.Values{}, &state); err != nil {
		return State{}, err
	}

	if !state.Online() && state.Error != "not_online_error" {
		return State{}, &GatewayError{Code: state.Error, Message: state.ErrorMessage}
	}
	if net.ParseIP(state.IP) == nil {
		return State{}, errors.New("gateway returned an invalid online_ip")
	}

	return state, nil
}

// Login authenticates unless the gateway already reports an online session.
// Credentials are used exactly as supplied and are never retained by Client.
func (c *Client) Login(ctx context.Context, username, password string) (PortalResponse, error) {
	return c.login(ctx, username, password, false)
}

// ForceLogin sends authentication even if the gateway reports an online session.
func (c *Client) ForceLogin(ctx context.Context, username, password string) (PortalResponse, error) {
	return c.login(ctx, username, password, true)
}

func (c *Client) login(ctx context.Context, username, password string, force bool) (PortalResponse, error) {
	if username == "" || password == "" {
		return PortalResponse{}, errors.New("username and password are required")
	}

	state, err := c.Status(ctx)
	if err != nil {
		return PortalResponse{}, err
	}
	if state.Online() && !force {
		return PortalResponse{}, ErrAlreadyOnline
	}

	ip := c.requestIP(state)
	acid, err := c.discoverACID(ctx)
	if err != nil {
		return PortalResponse{}, err
	}

	var challenge struct {
		Token   string `json:"challenge"`
		Error   string `json:"error"`
		Message string `json:"error_msg"`
	}
	if err := c.get(ctx, "/cgi-bin/get_challenge", url.Values{"username": {username}, "ip": {ip}}, &challenge); err != nil {
		return PortalResponse{}, err
	}
	if challenge.Error != "" && challenge.Error != "ok" {
		return PortalResponse{}, &GatewayError{Code: challenge.Error, Message: challenge.Message}
	}
	if challenge.Token == "" {
		return PortalResponse{}, errors.New("gateway returned an empty challenge")
	}

	info := loginInfo(username, password, ip, acid, challenge.Token)
	hash := passwordHash(challenge.Token)
	params := url.Values{
		"action":   {"login"},
		"username": {username},
		"password": {"{MD5}" + hash},
		"ac_id":    {acid},
		"ip":       {ip},
		"n":        {"200"},
		"type":     {"1"},
		"info":     {info},
		"chksum":   {checksum(challenge.Token, username, hash, acid, ip, info)},
	}

	return c.portalAction(ctx, "/cgi-bin/srun_portal", params)
}

// Logout disconnects the account reported by the gateway, using username only
// when the status has no username. When already offline, it returns an empty
// PortalResponse and nil without sending a logout request.
func (c *Client) Logout(ctx context.Context, username string) (PortalResponse, error) {
	return c.logout(ctx, username, false)
}

// ForceLogout also sends the logout request when the gateway reports offline.
func (c *Client) ForceLogout(ctx context.Context, username string) (PortalResponse, error) {
	return c.logout(ctx, username, true)
}

func (c *Client) logout(ctx context.Context, username string, force bool) (PortalResponse, error) {
	state, err := c.Status(ctx)
	if err != nil {
		return PortalResponse{}, err
	}
	if !state.Online() && !force {
		return PortalResponse{}, nil
	}

	if state.Username != "" {
		username = state.Username
	}
	if username == "" {
		return PortalResponse{}, errors.New("username is required for logout")
	}

	ip := c.requestIP(state)
	params := url.Values{"username": {username}, "ip": {ip}}
	if c.dumbTerminal {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		params.Set("time", timestamp)
		params.Set("unbind", "1")
		params.Set("sign", logoutSign(timestamp, username, ip))

		return c.portalAction(ctx, "/cgi-bin/rad_user_dm", params)
	}

	acid, err := c.discoverACID(ctx)
	if err != nil {
		return PortalResponse{}, err
	}
	params.Set("action", "logout")
	params.Set("ac_id", acid)

	return c.portalAction(ctx, "/cgi-bin/srun_portal", params)
}

func (c *Client) requestIP(state State) string {
	if c.ip.IsValid() {
		return c.ip.String()
	}

	return state.IP
}

func (c *Client) portalAction(ctx context.Context, path string, params url.Values) (PortalResponse, error) {
	var response PortalResponse
	if err := c.get(ctx, path, params, &response); err != nil {
		return response, err
	}

	if response.Error == "ok" || (response.Error == "" && response.Success != nil) {
		return response, nil
	}
	if response.Error == "" {
		return response, errors.New("gateway returned no authentication result")
	}

	return response, &GatewayError{Code: response.Error, Message: response.ErrorMessage}
}

func (c *Client) discoverACID(ctx context.Context) (string, error) {
	var lastErr error
	for _, endpoint := range []*url.URL{c.captive, c.portal} {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return "", err
		}

		response, err := c.http.Do(request)
		if err != nil {
			lastErr = transportError(err)
			continue
		}
		response.Body.Close()

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			lastErr = fmt.Errorf("HTTP %d", response.StatusCode)
			continue
		}
		if acid := response.Request.URL.Query().Get("ac_id"); acid != "" {
			return acid, nil
		}
		lastErr = errors.New("redirect did not contain ac_id")
	}

	return "", fmt.Errorf("discover gateway ac_id: %w", lastErr)
}

func (c *Client) get(ctx context.Context, path string, params url.Values, result any) error {
	params.Set("callback", "jsonp")

	endpoint := *c.portal
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	endpoint.RawQuery = params.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("%s: %w", path, transportError(err))
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s: HTTP %d", path, response.StatusCode)
	}

	const limit = 1 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(body) > limit {
		return fmt.Errorf("%s: response exceeds 1 MiB", path)
	}

	framed := strings.TrimSpace(string(body))
	framed = strings.TrimSuffix(framed, ";")
	if !strings.HasPrefix(framed, "jsonp(") || !strings.HasSuffix(framed, ")") {
		return fmt.Errorf("%s: invalid JSONP response", path)
	}

	if err := json.Unmarshal([]byte(framed[6:len(framed)-1]), result); err != nil {
		return fmt.Errorf("%s: invalid JSON response", path)
	}

	return nil
}

// net/http errors include the full query URL, which contains authentication data.
func transportError(err error) error {
	for {
		var requestError *url.Error
		if !errors.As(err, &requestError) {
			return err
		}

		err = requestError.Err
	}
}
