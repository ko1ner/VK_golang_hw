package movie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	ErrNotFound        = errors.New("movie not found")
	ErrUpstreamTimeout = errors.New("movie stand timeout")
)

const (
	defaultTimeout = 5 * time.Second
	// RangeTimeout ограничивает суммарное время загрузки диапазона,
	// чтобы запрос на большой [from; to] не висел бесконечно.
	RangeTimeout = 30 * time.Second
	maxMovieID   = 99_999
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
		timeout:    defaultTimeout,
	}
}

func (c *Client) GetByID(ctx context.Context, id int) (*Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/%d/info.0.json", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ErrUpstreamTimeout
		}
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("movie stand returned status %d", resp.StatusCode)
	}

	var mov Movie
	if err := json.NewDecoder(resp.Body).Decode(&mov); err != nil {
		return nil, err
	}
	return &mov, nil
}

// FindLast ищет последний существующий фильм, предполагая, что ID идут
// подряд от 1 до maxMovieID. Используется бинарный поиск, чтобы не делать
// десятки тысяч запросов.
func (c *Client) FindLast(ctx context.Context) (*Movie, error) {
	lo, hi := 1, maxMovieID
	last := 0

	for lo <= hi {
		mid := lo + (hi-lo)/2

		_, err := c.GetByID(ctx, mid)
		switch {
		case err == nil:
			last = mid
			lo = mid + 1
		case errors.Is(err, ErrNotFound):
			hi = mid - 1
		default:
			return nil, err
		}
	}

	if last == 0 {
		return nil, ErrNotFound
	}
	return c.GetByID(ctx, last)
}
