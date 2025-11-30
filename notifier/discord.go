package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/CY-Chang-tw/ClaudeUsageLimitChecker/api"
)

type DiscordNotifier struct {
	webhookURL string
	httpClient *http.Client
}

type DiscordWebhook struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

const (
	ColorGreen  = 0x00FF00
	ColorYellow = 0xFFFF00
	ColorOrange = 0xFFA500
	ColorRed    = 0xFF0000
)

func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *DiscordNotifier) SendUsageAlert(stats *api.UsageStats) error {
	// Use the primary period (7-day or 5-hour) for color coding
	color := d.getColorForPercentage(stats.UsagePercentage)
	emoji := d.getEmojiForPercentage(stats.UsagePercentage)

	// Build fields array
	fields := []DiscordEmbedField{}

	// Add 5-hour usage (Current session) if available
	if stats.HasFiveHour {
		fields = append(fields, DiscordEmbedField{
			Name:   "📊 Current Session (5-hour)",
			Value:  fmt.Sprintf("**%.1f%%**", stats.FiveHourUsage),
			Inline: true,
		})
		if stats.FiveHourResetsAt != nil {
			fields = append(fields, DiscordEmbedField{
				Name:   "⏰ Resets At",
				Value:  stats.FiveHourResetsAt.Format("Jan 02, 15:04 MST"),
				Inline: true,
			})
		}
		// Add empty field for spacing
		fields = append(fields, DiscordEmbedField{
			Name:   "\u200b",
			Value:  "\u200b",
			Inline: true,
		})
	}

	// Add 7-day usage (Weekly limits) if available
	if stats.HasSevenDay {
		fields = append(fields, DiscordEmbedField{
			Name:   "📈 Weekly Limits (7-day)",
			Value:  fmt.Sprintf("**%.1f%%**", stats.SevenDayUsage),
			Inline: true,
		})
		if stats.SevenDayResetsAt != nil {
			fields = append(fields, DiscordEmbedField{
				Name:   "⏰ Resets At",
				Value:  stats.SevenDayResetsAt.Format("Jan 02, 15:04 MST"),
				Inline: true,
			})
		}
	}

	embed := DiscordEmbed{
		Title:       fmt.Sprintf("%s Claude Usage Alert", emoji),
		Description: d.getAlertMessage(stats.UsagePercentage),
		Color:       color,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: fmt.Sprintf("Primary Monitor: %s", stats.PeriodType),
		},
	}

	webhook := DiscordWebhook{
		Embeds: []DiscordEmbed{embed},
	}

	return d.sendWebhook(webhook)
}

func (d *DiscordNotifier) SendTestNotification() error {
	embed := DiscordEmbed{
		Title:       "✅ Test Notification",
		Description: "Claude Usage Monitor is working correctly!",
		Color:       ColorGreen,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: "Claude Usage Monitor",
		},
	}

	webhook := DiscordWebhook{
		Embeds: []DiscordEmbed{embed},
	}

	return d.sendWebhook(webhook)
}

func (d *DiscordNotifier) sendWebhook(webhook DiscordWebhook) error {
	jsonData, err := json.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook: %w", err)
	}

	resp, err := d.httpClient.Post(d.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	return nil
}

func (d *DiscordNotifier) getColorForPercentage(percentage float64) int {
	switch {
	case percentage >= 95:
		return ColorRed
	case percentage >= 90:
		return ColorOrange
	case percentage >= 80:
		return ColorYellow
	default:
		return ColorGreen
	}
}

func (d *DiscordNotifier) getEmojiForPercentage(percentage float64) string {
	switch {
	case percentage >= 95:
		return "🔴"
	case percentage >= 90:
		return "🟠"
	case percentage >= 80:
		return "🟡"
	default:
		return "🟢"
	}
}

func (d *DiscordNotifier) getAlertMessage(percentage float64) string {
	switch {
	case percentage >= 95:
		return "⚠️ **CRITICAL**: Your Claude usage has exceeded 95%! You're almost at your limit."
	case percentage >= 90:
		return "⚠️ **WARNING**: Your Claude usage has exceeded 90%. Consider monitoring your usage closely."
	case percentage >= 80:
		return "⚠️ **NOTICE**: Your Claude usage has exceeded 80%. You may want to keep track of your consumption."
	default:
		return "Your Claude usage is within normal limits."
	}
}
