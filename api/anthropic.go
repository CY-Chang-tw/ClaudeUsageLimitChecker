package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	sessionKey string
	orgID      string
	httpClient *http.Client
}

type UsageResponse struct {
	FiveHour    *UsagePeriod `json:"five_hour"`
	SevenDay    *UsagePeriod `json:"seven_day"`
	SevenDayOAuthApps *UsagePeriod `json:"seven_day_oauth_apps"`
}

type UsagePeriod struct {
	Utilization float64    `json:"utilization"`
	ResetsAt    *time.Time `json:"resets_at"`
}

type UsageStats struct {
	CurrentUsage    float64
	UsagePercentage float64
	PeriodType      string
	ResetsAt        time.Time
	LastChecked     time.Time
}

func NewClient(sessionKey, orgID string) *Client {
	return &Client{
		sessionKey: sessionKey,
		orgID:      orgID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) GetUsage() (*UsageStats, error) {
	url := fmt.Sprintf("https://claude.ai/api/organizations/%s/usage", c.orgID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", c.sessionKey))
	req.Header.Set("anthropic-client-platform", "web_claude_ai")
	req.Header.Set("anthropic-client-version", "1.0.0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://claude.ai/settings/usage")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var usageResp UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Prioritize seven_day over five_hour
	stats := &UsageStats{
		LastChecked: time.Now(),
	}

	if usageResp.SevenDay != nil {
		stats.UsagePercentage = usageResp.SevenDay.Utilization
		stats.PeriodType = "7-day"
		if usageResp.SevenDay.ResetsAt != nil {
			stats.ResetsAt = *usageResp.SevenDay.ResetsAt
		}
	} else if usageResp.FiveHour != nil {
		stats.UsagePercentage = usageResp.FiveHour.Utilization
		stats.PeriodType = "5-hour"
		if usageResp.FiveHour.ResetsAt != nil {
			stats.ResetsAt = *usageResp.FiveHour.ResetsAt
		}
	} else {
		return nil, fmt.Errorf("no usage data available")
	}

	stats.CurrentUsage = stats.UsagePercentage

	return stats, nil
}
