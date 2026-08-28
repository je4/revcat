package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"emperror.dev/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/utils/v2/pkg/zLogger"
)

// Client is the interface for interacting with the RevCat REST (Swagger) API.
type Client interface {
	GetItem(ctx context.Context, signature string) (*sourcetype.SourceData, error)
	UpdateItem(ctx context.Context, signature string, data *sourcetype.SourceData) error
	DeleteItem(ctx context.Context, signature string) error
	GetSignature(ctx context.Context, signature string) (*sourcetype.SourceData, error)
	UpdateSignature(ctx context.Context, signature string, data *sourcetype.SourceData) error
	DeleteSignature(ctx context.Context, signature string) error
}

type client struct {
	baseURL        *url.URL
	httpClient     *http.Client
	jwtKey         string
	jwtDuration    time.Duration
	bearerToken    string
	tokenProvider  func() (string, error)
	cachedToken    string
	tokenExpiresAt time.Time
	tokenMu        sync.Mutex
	userAgent      string
	headers        map[string]string
	logger         zLogger.ZLogger
}

var _ Client = (*client)(nil)

// New creates a new RevCat REST API client.
func New(rawBaseURL string, opts ...Option) (Client, error) {
	if rawBaseURL == "" {
		return nil, errors.New("baseURL cannot be empty")
	}

	if !strings.HasPrefix(rawBaseURL, "http://") && !strings.HasPrefix(rawBaseURL, "https://") {
		rawBaseURL = "http://" + rawBaseURL
	}

	parsedURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid baseURL %q", rawBaseURL)
	}

	c := &client{
		baseURL:     parsedURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		jwtDuration: 1 * time.Hour,
		headers:     make(map[string]string),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// NewClient is an alias for New.
func NewClient(rawBaseURL string, opts ...Option) (Client, error) {
	return New(rawBaseURL, opts...)
}

func (c *client) itemURL(signature string) (string, error) {
	basePath := strings.TrimSuffix(c.baseURL.Path, "/")
	if !strings.HasSuffix(basePath, "/rest") {
		basePath = basePath + "/rest"
	}
	itemPath := basePath + "/item/" + url.PathEscape(signature)

	u := *c.baseURL
	u.Path = itemPath
	return u.String(), nil
}

func (c *client) getToken() (string, error) {
	if c.bearerToken != "" {
		return c.bearerToken, nil
	}

	if c.tokenProvider != nil {
		return c.tokenProvider()
	}

	if c.jwtKey != "" {
		c.tokenMu.Lock()
		defer c.tokenMu.Unlock()

		now := time.Now()
		if c.cachedToken != "" && now.Add(10*time.Second).Before(c.tokenExpiresAt) {
			return c.cachedToken, nil
		}

		duration := c.jwtDuration
		if duration <= 0 {
			duration = 1 * time.Hour
		}
		expiresAt := now.Add(duration)
		claims := &jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(c.jwtKey))
		if err != nil {
			return "", errors.Wrap(err, "failed to sign JWT token")
		}

		c.cachedToken = signedToken
		c.tokenExpiresAt = expiresAt
		return signedToken, nil
	}

	return "", nil
}

func (c *client) applyHeaders(req *http.Request) error {
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	token, err := c.getToken()
	if err != nil {
		return errors.Wrap(err, "failed to get authorization token")
	}
	if token != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func (c *client) parseResponseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var msg string
	var jsonErr struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &jsonErr); err == nil && jsonErr.Error != "" {
		msg = jsonErr.Error
	} else {
		msg = strings.TrimSpace(string(body))
	}

	if c.logger != nil {
		c.logger.Error().Msgf("rest client http error %d (%s): %s", resp.StatusCode, resp.Status, msg)
	}

	return &HTTPError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Message:    msg,
		Body:       body,
	}
}

// GetItem retrieves a catalog entry by its signature.
func (c *client) GetItem(ctx context.Context, signature string) (*sourcetype.SourceData, error) {
	if signature == "" {
		return nil, ErrEmptySignature
	}

	reqURL, err := c.itemURL(signature)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct item URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	req.Header.Set("Accept", "application/json")
	if err := c.applyHeaders(req); err != nil {
		return nil, err
	}

	if c.logger != nil {
		c.logger.Debug().Msgf("GET %s", reqURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute GET %s", reqURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseResponseError(resp)
	}

	var data sourcetype.SourceData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.Wrap(err, "failed to decode response JSON")
	}

	return &data, nil
}

// UpdateItem creates or updates a catalog entry by its signature.
func (c *client) UpdateItem(ctx context.Context, signature string, data *sourcetype.SourceData) error {
	if signature == "" {
		return ErrEmptySignature
	}
	if data == nil {
		return ErrNilData
	}

	reqURL, err := c.itemURL(signature)
	if err != nil {
		return errors.Wrap(err, "failed to construct item URL")
	}

	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal item data")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/plain, application/json")
	if err := c.applyHeaders(req); err != nil {
		return err
	}

	if c.logger != nil {
		c.logger.Debug().Msgf("POST %s", reqURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to execute POST %s", reqURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseResponseError(resp)
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// DeleteItem deletes a catalog entry by its signature.
func (c *client) DeleteItem(ctx context.Context, signature string) error {
	if signature == "" {
		return ErrEmptySignature
	}

	reqURL, err := c.itemURL(signature)
	if err != nil {
		return errors.Wrap(err, "failed to construct item URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}

	req.Header.Set("Accept", "text/plain, application/json")
	if err := c.applyHeaders(req); err != nil {
		return err
	}

	if c.logger != nil {
		c.logger.Debug().Msgf("DELETE %s", reqURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to execute DELETE %s", reqURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseResponseError(resp)
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// GetSignature is an alias for GetItem.
func (c *client) GetSignature(ctx context.Context, signature string) (*sourcetype.SourceData, error) {
	return c.GetItem(ctx, signature)
}

// UpdateSignature is an alias for UpdateItem.
func (c *client) UpdateSignature(ctx context.Context, signature string, data *sourcetype.SourceData) error {
	return c.UpdateItem(ctx, signature, data)
}

// DeleteSignature is an alias for DeleteItem.
func (c *client) DeleteSignature(ctx context.Context, signature string) error {
	return c.DeleteItem(ctx, signature)
}
