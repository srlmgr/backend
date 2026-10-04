package irdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/support/cache"
)

type (
	Option        func(*config)
	TokenProvider func() (string, error)
	config        struct {
		ctx   context.Context
		tp    TokenProvider
		cache cache.Cache
	}
	RateLimit struct {
		Limit     int
		Remaining int
		Reset     time.Time
	}

	IrData struct {
		cfg      config
		client   *retryablehttp.Client
		s3Client *retryablehttp.Client
		rlMutex  sync.Mutex
		baseURL  *url.URL
	}
	s3Link struct {
		Link    string    `json:"link"`
		Expires time.Time `json:"expires"`
	}
	//nolint:tagliatelle // external definition
	chunkInfo struct {
		ChunkSize       int      `json:"chunk_size"`
		NumChunks       int      `json:"num_chunks"`
		Rows            int      `json:"rows"`
		BaseDownloadURL string   `json:"base_download_url"`
		ChunkFileNames  []string `json:"chunk_file_names"`
	}

	// construct to provide resolved chunk data to caller
	//nolint:tagliatelle // by design.
	ChunkData[E any] struct {
		Data []E `json:"_chunk_data"`
	}
	ctxKey struct{}
)

const baseURL = "https://members-ng.iracing.com/data"

var (
	ErrNoTokenProvider = fmt.Errorf("no token provider configured")
	ctxKeyInstance     = &ctxKey{}
)

func AddToContext(ctx context.Context, i *IrData) context.Context {
	return context.WithValue(ctx, ctxKeyInstance, i)
}

func FromContext(ctx context.Context) (*IrData, bool) {
	i, ok := ctx.Value(ctxKeyInstance).(*IrData)
	return i, ok
}

func NewIrData(opts ...Option) (*IrData, error) {
	cfg := config{
		ctx:   context.Background(),
		tp:    func() (string, error) { return "", ErrNoTokenProvider },
		cache: cache.NewNoopCache(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	client := retryablehttp.NewClient()
	client.Logger = newCustomLeveledLogger(log.Default().Named("iracing.api"))
	s3Client := retryablehttp.NewClient()
	s3Client.Logger = newCustomLeveledLogger(log.Default().Named("iracing.s3"))
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &IrData{
		cfg:      cfg,
		client:   client,
		s3Client: s3Client, rlMutex: sync.Mutex{}, baseURL: parsedBaseURL,
	}, nil
}

func WithContext(ctx context.Context) Option {
	return func(c *config) {
		c.ctx = ctx
	}
}

func WithTokenProvider(tp TokenProvider) Option {
	return func(c *config) {
		c.tp = tp
	}
}

func WithCache(arg cache.Cache) Option {
	return func(c *config) {
		c.cache = arg
	}
}

//nolint:funlen // much to do here
func (i *IrData) Get(uri string) ([]byte, error) {
	if b, ok := i.cfg.cache.Get(uri); ok {
		return b, nil
	}
	token, err := i.cfg.tp()
	if err != nil {
		return nil, err
	}

	uriRef, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}
	reqURL := i.baseURL.ResolveReference(uriRef)

	req, err := retryablehttp.NewRequestWithContext(
		i.cfg.ctx,
		http.MethodGet, reqURL.String(), http.NoBody,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	log.Debug(
		"response received",
		log.Int("status", resp.StatusCode),
		log.String("rate-limit", resp.Header.Get("X-RateLimit-Limit")),
		log.String("rate-remaining", resp.Header.Get("X-RateLimit-Remaining")),
		log.String("rate-reset", resp.Header.Get("X-RateLimit-Reset")),
	)
	defer resp.Body.Close()
	rateReset, _ := strconv.ParseFloat(resp.Header.Get("X-RateLimit-Reset"), 64)
	if rateReset > 0 {
		log.Debug("rate limit reset time",
			log.String("reset-time", time.Unix(int64(rateReset), 0).String()))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var s3link s3Link
	//nolint:nestif // by design
	if err := json.Unmarshal(body, &s3link); err == nil {
		s3Resp, err := i.s3Client.Get(s3link.Link)
		if err != nil {
			return nil, err
		}
		defer s3Resp.Body.Close()
		if s3Resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code from s3 link: %d",
				s3Resp.StatusCode)
		}
		body, err = io.ReadAll(s3Resp.Body)
		if err != nil {
			return nil, err
		}
		body, err = i.resolveChunks(body)
		if err != nil {
			return nil, err
		}
	}
	if cacheErr := i.cfg.cache.Set(uri, body); cacheErr != nil {
		log.Warn("failed to set cache", log.ErrorField(cacheErr))
	}
	return body, nil
}

func (i *IrData) resolveChunks(body []byte) ([]byte, error) {
	var ci chunkInfo
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return body, err
	}

	if raw["chunk_info"] == nil {
		return body, nil
	}
	if ciStr, err := json.Marshal(raw["chunk_info"]); err == nil {
		if err := json.Unmarshal(ciStr, &ci); err != nil {
			return body, err
		}
	} else {
		return body, nil
	}
	if len(ci.ChunkFileNames) == 0 {
		return body, nil
	}

	var results []any
	for _, chunk := range ci.ChunkFileNames {
		chunkBody, err := i.readChunk(fmt.Sprintf("%s%s", ci.BaseDownloadURL, chunk))
		if err != nil {
			return nil, err
		}
		var chunkResult []any
		if err := json.Unmarshal(chunkBody, &chunkResult); err == nil {
			results = append(results, chunkResult...)
		} else {
			log.Warn("failed to unmarshal chunk", log.ErrorField(err))
		}
	}
	if len(results) == 0 {
		return body, nil
	}

	raw["_chunk_data"] = results
	combined, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return combined, nil
}

func (i *IrData) readChunk(uri string) ([]byte, error) {
	s3Resp, err := i.s3Client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer s3Resp.Body.Close()
	if s3Resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected status code from s3 chunk link: %d",
			s3Resp.StatusCode,
		)
	}
	chunkBody, err := io.ReadAll(s3Resp.Body)
	if err != nil {
		return nil, err
	}
	return chunkBody, nil
}
