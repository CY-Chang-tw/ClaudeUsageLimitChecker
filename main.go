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

type Monitor struct {
	config         *config.Config
	apiClient      *api.Client
	discordNotifier *notifier.DiscordNotifier
	lastNotified   map[float64]time.Time
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
		lastNotified:    make(map[float64]time.Time),
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
	for _, threshold := range m.config.WarningLevels {
		if stats.UsagePercentage >= threshold {
			// Check if we already notified for this threshold recently
			if m.shouldNotify(threshold) {
				log.Printf("Usage exceeded %.0f%% threshold, sending notification...", threshold)

				err := m.discordNotifier.SendUsageAlert(stats)

				if err != nil {
					log.Printf("Failed to send Discord notification: %v", err)
				} else {
					log.Printf("Notification sent successfully for %.0f%% threshold", threshold)
					m.lastNotified[threshold] = time.Now()
				}
			}

			// Only notify for the highest exceeded threshold
			break
		}
	}
}

func (m *Monitor) shouldNotify(threshold float64) bool {
	lastTime, exists := m.lastNotified[threshold]
	if !exists {
		return true
	}

	// Don't send notifications more than once per hour for the same threshold
	cooldownPeriod := 1 * time.Hour
	return time.Since(lastTime) > cooldownPeriod
}

func init() {
	// Set up logging format
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
}
