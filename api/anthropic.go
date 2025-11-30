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
	CurrentUsage      float64
	UsagePercentage   float64
	PeriodType        string
	ResetsAt          time.Time
	LastChecked       time.Time
	FiveHourUsage     float64
	FiveHourResetsAt  *time.Time
	SevenDayUsage     float64
	SevenDayResetsAt  *time.Time
	HasFiveHour       bool
	HasSevenDay       bool
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

	// Capture both periods
	stats := &UsageStats{
		LastChecked: time.Now(),
	}

	// Capture 5-hour data
	if usageResp.FiveHour != nil {
		stats.FiveHourUsage = usageResp.FiveHour.Utilization
		stats.FiveHourResetsAt = usageResp.FiveHour.ResetsAt
		stats.HasFiveHour = true
	}

	// Capture 7-day data
	if usageResp.SevenDay != nil {
		stats.SevenDayUsage = usageResp.SevenDay.Utilization
		stats.SevenDayResetsAt = usageResp.SevenDay.ResetsAt
		stats.HasSevenDay = true
	}

	// Prioritize seven_day over five_hour for primary metric
	if stats.HasSevenDay {
		stats.UsagePercentage = stats.SevenDayUsage
		stats.PeriodType = "7-day"
		if stats.SevenDayResetsAt != nil {
			stats.ResetsAt = *stats.SevenDayResetsAt
		}
	} else if stats.HasFiveHour {
		stats.UsagePercentage = stats.FiveHourUsage
		stats.PeriodType = "5-hour"
		if stats.FiveHourResetsAt != nil {
			stats.ResetsAt = *stats.FiveHourResetsAt
		}
	} else {
		return nil, fmt.Errorf("no usage data available")
	}

	stats.CurrentUsage = stats.UsagePercentage

	return stats, nil
}
