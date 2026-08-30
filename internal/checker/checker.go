package checker

import (
	"context"
	"net/http"
	"time"

	"isitdown/internal/store"
)

type Result struct {
	Status         string
	StatusCode     *int
	ResponseTimeMS *int
	Error          *string
}

// Check performs a single HTTP GET against url, bounded by timeout.
func Check(ctx context.Context, url string, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		errMsg := err.Error()
		return Result{Status: "down", Error: &errMsg}
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	elapsedMS := int(time.Since(start).Milliseconds())

	if err != nil {
		errMsg := err.Error()
		return Result{Status: "down", ResponseTimeMS: &elapsedMS, Error: &errMsg}
	}
	defer resp.Body.Close()

	status := "up"
	if resp.StatusCode >= 400 {
		status = "down"
	}
	code := resp.StatusCode

	return Result{Status: status, StatusCode: &code, ResponseTimeMS: &elapsedMS}
}

func (r Result) ToCheck(monitorID string) store.Check {
	return store.Check{
		MonitorID:      monitorID,
		Status:         r.Status,
		StatusCode:     r.StatusCode,
		ResponseTimeMS: r.ResponseTimeMS,
		Error:          r.Error,
	}
}
