package client

import (
	"net/http"
	"time"

	"github.com/je4/utils/v2/pkg/zLogger"
)

type Option func(*client)

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithJWTKey configures automatic JWT generation using the provided HMAC secret key.
func WithJWTKey(key string) Option {
	return func(c *client) {
		c.jwtKey = key
	}
}

// WithJWTDuration sets the validity duration of automatically generated JWT tokens.
func WithJWTDuration(d time.Duration) Option {
	return func(c *client) {
		if d > 0 {
			c.jwtDuration = d
		}
	}
}

// WithBearerToken sets a static Bearer token for authorization.
func WithBearerToken(token string) Option {
	return func(c *client) {
		c.bearerToken = token
	}
}

// WithTokenProvider sets a dynamic token provider callback.
func WithTokenProvider(provider func() (string, error)) Option {
	return func(c *client) {
		c.tokenProvider = provider
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *client) {
		c.userAgent = ua
	}
}

// WithHeader sets an additional HTTP header sent with all requests.
func WithHeader(key, value string) Option {
	return func(c *client) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	}
}

// WithLogger sets an optional zLogger.ZLogger instance.
func WithLogger(logger zLogger.ZLogger) Option {
	return func(c *client) {
		c.logger = logger
	}
}

// WithTimeout sets a timeout on the default http.Client.
func WithTimeout(timeout time.Duration) Option {
	return func(c *client) {
		if c.httpClient != nil {
			c.httpClient.Timeout = timeout
		}
	}
}
