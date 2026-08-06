// internal/client/kotodb_client.go
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

var ErrItemNotFound = errors.New("item not found")
var ErrUpstreamError = errors.New("upstream error")

type KotoDBClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewKotoDBClient(baseURL, apiKey string) *KotoDBClient {
	return &KotoDBClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

type SearchByModelNumberResponse struct {
	Status string `json:"status"`
	Data   struct {
		ID                    int64   `json:"id"`
		ItemName              string  `json:"item_name"`
		ModelNumber           string  `json:"model_number"`
		ImageURL              *string `json:"image_url"`
		RegularPrice          *int64  `json:"regular_price"`
		ReleaseDate           *string `json:"release_date"`
		MercariURL            *string `json:"mercari_url"`
		MarketPrice           *int64  `json:"market_price"`
		MarketPriceRecordedAt *string `json:"market_price_recorded_at"`
	} `json:"data"`
	Message string `json:"message"`
}

func (c *KotoDBClient) SearchByModelNumber(ctx context.Context, modelNumber string) (*SearchByModelNumberResponse, error) {

	u, err := url.Parse(c.BaseURL + "/api/v1/internal/items/search_by_model_number")

	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("model_number", modelNumber)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.APIKey != "" {
		req.Header.Set("X-Internal-API-Key", c.APIKey)
	}

	// T5：Rails APIへのHTTP通信開始
	t5StartedAt := time.Now()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		t5Ms := float64(time.Since(t5StartedAt).Microseconds()) / 1000

		log.Printf(
			"[OCR_MEASUREMENT] model_number=%s T5_ms=%.3f status=request_failed error=%v",
			modelNumber,
			t5Ms,
			err,
		)

		return nil, err
	}
	defer resp.Body.Close()

	var result SearchByModelNumberResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t5Ms := float64(time.Since(t5StartedAt).Microseconds()) / 1000

		log.Printf(
			"[OCR_MEASUREMENT] model_number=%s T5_ms=%.3f status=decode_failed http_status=%d error=%v",
			modelNumber,
			t5Ms,
			resp.StatusCode,
			err,
		)

		return nil, err
	}

	// T5：RailsレスポンスのJSON解析完了
	t5Ms := float64(time.Since(t5StartedAt).Microseconds()) / 1000

	log.Printf(
		"[OCR_MEASUREMENT] model_number=%s T5_ms=%.3f http_status=%d",
		modelNumber,
		t5Ms,
		resp.StatusCode,
	)

	switch resp.StatusCode {
	case http.StatusOK:
		return &result, nil
	case http.StatusNotFound:
		return nil, ErrItemNotFound
	default:
		return nil, fmt.Errorf("%w: status=%d message=%s", ErrUpstreamError, resp.StatusCode, result.Message)
	}
}
