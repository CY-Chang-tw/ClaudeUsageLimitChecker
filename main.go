package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CY-Chang-tw/ClaudeUsageLimitChecker/api"
	"github.com/CY-Chang-tw/ClaudeUsageLimitChecker/config"
	"github.com/CY-Chang-tw/ClaudeUsageLimitChecker/notifier"
)

type ThresholdKey struct {
	Threshold float64
	Period    string  // "5-hour", "7-day", or "both"
}

type Monitor struct {
	config         *config.Config
	apiClient      *api.Client
	discordNotifier *notifier.DiscordNotifier
	lastNotified   map[ThresholdKey]time.Time
}

func main() {
	log.Println("Starting Claude Usage Monitor...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize monitor
	monitor := &Monitor{
		config:          cfg,
		apiClient:       api.NewClient(cfg.ClaudeSessionKey, cfg.ClaudeOrgID),
		discordNotifier: notifier.NewDiscordNotifier(cfg.DiscordWebhookURL),
		lastNotified:    make(map[ThresholdKey]time.Time),
	}

	// Send test notification
	log.Println("Sending test notification to Discord...")
	if err := monitor.discordNotifier.SendTestNotification(); err != nil {
		log.Printf("Warning: Failed to send test notification: %v", err)
	} else {
		log.Println("Test notification sent successfully!")
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping monitor...")
		cancel()
	}()

	// Start monitoring loop
	monitor.start(ctx)

	log.Println("Claude Usage Monitor stopped.")
}

func (m *Monitor) start(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.config.CheckInterval) * time.Minute)
	defer ticker.Stop()

	// Run first check immediately
	m.checkUsage(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkUsage(ctx)
		}
	}
}

func (m *Monitor) checkUsage(ctx context.Context) {
	log.Println("Checking Claude usage...")

	stats, err := m.apiClient.GetUsage()
	if err != nil {
		log.Printf("Error getting usage: %v", err)
		return
	}

	// Log both periods if available
	if stats.HasFiveHour && stats.HasSevenDay {
		log.Printf("Usage Stats - 5-hour: %.1f%%, 7-day: %.1f%% (Primary: %s)",
			stats.FiveHourUsage, stats.SevenDayUsage, stats.PeriodType)
	} else if stats.HasFiveHour {
		log.Printf("Usage Stats - 5-hour: %.1f%%", stats.FiveHourUsage)
	} else if stats.HasSevenDay {
		log.Printf("Usage Stats - 7-day: %.1f%%", stats.SevenDayUsage)
	}

	// Check if we need to send notifications
	m.checkThresholds(stats)
}

func (m *Monitor) checkThresholds(stats *api.UsageStats) {
	// Combine both threshold lists and remove duplicates
	thresholdMap := make(map[float64]bool)

	// Add 5-hour thresholds
	if stats.HasFiveHour {
		for _, t := range m.config.FiveHourWarningLevels {
			thresholdMap[t] = true
		}
	}

	// Add 7-day thresholds
	if stats.HasSevenDay {
		for _, t := range m.config.SevenDayWarningLevels {
			thresholdMap[t] = true
		}
	}

	// Convert to sorted slice
	thresholds := make([]float64, 0, len(thresholdMap))
	for t := range thresholdMap {
		thresholds = append(thresholds, t)
	}

	// Sort in descending order (check highest first)
	for i := 0; i < len(thresholds); i++ {
		for j := i + 1; j < len(thresholds); j++ {
			if thresholds[i] < thresholds[j] {
				thresholds[i], thresholds[j] = thresholds[j], thresholds[i]
			}
		}
	}

	// Check each threshold
	for _, threshold := range thresholds {
		// Check if this threshold applies to 5-hour and if it's exceeded
		fiveHourApplies := stats.HasFiveHour && containsThreshold(m.config.FiveHourWarningLevels, threshold)
		fiveHourExceeds := fiveHourApplies && stats.FiveHourUsage >= threshold

		// Check if this threshold applies to 7-day and if it's exceeded
		sevenDayApplies := stats.HasSevenDay && containsThreshold(m.config.SevenDayWarningLevels, threshold)
		sevenDayExceeds := sevenDayApplies && stats.SevenDayUsage >= threshold

		if fiveHourExceeds || sevenDayExceeds {
			// Determine the period for this threshold
			var period string
			if fiveHourExceeds && sevenDayExceeds {
				period = "both"
			} else if fiveHourExceeds {
				period = "5-hour"
			} else {
				period = "7-day"
			}

			// Create threshold key for cooldown tracking
			key := ThresholdKey{
				Threshold: threshold,
				Period:    period,
			}

			// Check if we already notified for this threshold+period combination recently
			if m.shouldNotify(key) {
				log.Printf("Usage exceeded %.0f%% threshold (%s), sending notification...", threshold, period)

				err := m.discordNotifier.SendUsageAlert(stats)

				if err != nil {
					log.Printf("Failed to send Discord notification: %v", err)
				} else {
					log.Printf("Notification sent successfully for %.0f%% threshold (%s)", threshold, period)
					m.lastNotified[key] = time.Now()
				}
			}

			// Only notify for the highest exceeded threshold
			break
		}
	}
}

func containsThreshold(thresholds []float64, value float64) bool {
	for _, t := range thresholds {
		if t == value {
			return true
		}
	}
	return false
}

func (m *Monitor) shouldNotify(key ThresholdKey) bool {
	lastTime, exists := m.lastNotified[key]
	if !exists {
		return true
	}

	// Use different cooldown periods based on usage period type
	var cooldownPeriod time.Duration
	if key.Period == "5-hour" {
		// 5-hour period resets frequently, use shorter cooldown
		cooldownPeriod = 30 * time.Minute
	} else {
		// 7-day period and "both" use longer cooldown
		cooldownPeriod = 1 * time.Hour
	}

	return time.Since(lastTime) > cooldownPeriod
}

func init() {
	// Set up logging format
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
}
