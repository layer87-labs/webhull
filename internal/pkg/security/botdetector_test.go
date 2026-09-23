package security

import "testing"

func TestBotDetector_IsBot(t *testing.T) {
	d := NewBotDetector()

	tests := []struct {
		name      string
		userAgent string
		wantBot   bool
	}{
		// Search-engine and SEO crawlers (pre-existing).
		{"googlebot", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", true},
		{"bingbot", "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", true},
		{"ahrefs", "Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)", true},

		// Monitoring / uptime probes — the actual subject of #32.
		{"blackbox-exporter", "Blackbox-Exporter/0.28.0", true},
		{"uptimerobot", "Mozilla/5.0+(compatible; UptimeRobot/2.0; http://www.uptimerobot.com/)", true},
		{"kube-probe", "kube-probe/1.29", true},
		{"prometheus", "Prometheus/2.45.0", true},
		{"pingdom", "Pingdom.com_bot_version_1.4", true},
		{"statuscake", "StatusCake_Match_Beta", true},
		{"betteruptime", "Better Uptime Bot Version 1.0", true},
		{"site24x7", "Site24x7", true},

		// Generic HTTP client / scripting libraries.
		{"curl", "curl/8.4.0", true},
		{"wget", "Wget/1.21.3", true},
		{"go-http-client", "Go-http-client/1.1", true},
		{"python-requests", "python-requests/2.31.0", true},
		{"python-urllib", "Python-urllib/3.11", true},
		{"aiohttp", "Python/3.11 aiohttp/3.9.1", true},
		{"axios", "axios/1.6.0", true},
		{"node-fetch", "node-fetch/1.0 (+https://github.com/node-fetch/node-fetch)", true},
		{"okhttp", "okhttp/4.12.0", true},
		{"java httpclient", "Java/17.0.2", true},
		{"libwww", "libwww-perl/6.68", true},
		{"apache httpclient", "Apache-HttpClient/4.5.13 (Java/11.0.2)", true},

		// Headless / automated browsers.
		{"headlesschrome", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36", true},
		{"lighthouse", "Mozilla/5.0 (Linux; Android 7.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 Chrome-Lighthouse", true},

		// Empty User-Agent always counts as a bot.
		{"empty", "", true},

		// Real browsers must never be misclassified — "java" must not match
		// "javascript" and none of the client patterns should fire.
		{"chrome", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", false},
		{"firefox", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0", false},
		{"safari", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15", false},
		{"edge", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0", false},
		{"samsung", "Mozilla/5.0 (Linux; Android 13; SM-S911B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36", false},
		{"javascript-in-real-ua-still-safe", "Mozilla/5.0 JavaScriptEnabledBrowser/1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.IsBot(tt.userAgent); got != tt.wantBot {
				t.Errorf("IsBot(%q) = %v, want %v", tt.userAgent, got, tt.wantBot)
			}
		})
	}
}

func TestBotDetector_GooglebotVsHuman(t *testing.T) {
	d := NewBotDetector()

	if !d.IsBot("Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/W.X.Y.Z Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)") {
		t.Error("Googlebot smartphone UA should be detected as a bot")
	}
}
