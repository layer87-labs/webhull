package forms

import (
	"fmt"
	"strings"
	"unicode"
)

// SpamFilter provides configurable spam detection for text content.
type SpamFilter struct {
	spamKeywords   []string
	maxLinks       int
	maxUpperRatio  float64
	minWordVariety int
	bannedPhrases  []string
}

// NewSpamFilter creates a spam filter with sensible defaults.
func NewSpamFilter() *SpamFilter {
	return &SpamFilter{
		spamKeywords:   defaultSpamKeywords(),
		maxLinks:       2,
		maxUpperRatio:  0.5,
		minWordVariety: 3,
		bannedPhrases:  defaultBannedPhrases(),
	}
}

// ValidateContent runs all spam checks against the combined input.
func (sf *SpamFilter) ValidateContent(name, subject, message string) error {
	content := strings.ToLower(name + " " + subject + " " + message)

	if err := sf.checkSpamKeywords(content); err != nil {
		return err
	}
	if err := sf.checkLinkCount(content); err != nil {
		return err
	}
	if err := sf.checkUpperCaseRatio(message); err != nil {
		return err
	}
	if err := sf.checkWordVariety(message); err != nil {
		return err
	}
	if err := sf.checkBannedPhrases(content); err != nil {
		return err
	}
	if err := sf.checkRepeatedCharacters(message); err != nil {
		return err
	}
	return nil
}

// AddSpamKeywords adds additional keywords to the filter.
func (sf *SpamFilter) AddSpamKeywords(keywords []string) {
	sf.spamKeywords = append(sf.spamKeywords, keywords...)
}

// SetMaxLinks configures the maximum allowed link count.
func (sf *SpamFilter) SetMaxLinks(max int) {
	sf.maxLinks = max
}

func (sf *SpamFilter) checkSpamKeywords(content string) error {
	for _, keyword := range sf.spamKeywords {
		if strings.Contains(content, keyword) {
			return fmt.Errorf("message contains prohibited content")
		}
	}
	return nil
}

func (sf *SpamFilter) checkLinkCount(content string) error {
	count := strings.Count(content, "http://") + strings.Count(content, "https://") + strings.Count(content, "www.")
	if count > sf.maxLinks {
		return fmt.Errorf("too many links in message (max %d)", sf.maxLinks)
	}
	return nil
}

func (sf *SpamFilter) checkUpperCaseRatio(message string) error {
	if len(message) < 10 {
		return nil
	}
	upper, letters := 0, 0
	for _, r := range message {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if letters > 0 && float64(upper)/float64(letters) > sf.maxUpperRatio {
		return fmt.Errorf("too many uppercase characters")
	}
	return nil
}

func (sf *SpamFilter) checkWordVariety(message string) error {
	if len(message) < 50 {
		return nil
	}
	words := strings.Fields(strings.ToLower(message))
	unique := make(map[string]struct{})
	for _, w := range words {
		if len(w) > 2 {
			unique[w] = struct{}{}
		}
	}
	if len(unique) < sf.minWordVariety && len(words) > 10 {
		return fmt.Errorf("message has spam characteristics")
	}
	return nil
}

func (sf *SpamFilter) checkBannedPhrases(content string) error {
	for _, phrase := range sf.bannedPhrases {
		if strings.Contains(content, phrase) {
			return fmt.Errorf("message contains prohibited phrases")
		}
	}
	return nil
}

func (sf *SpamFilter) checkRepeatedCharacters(message string) error {
	maxRepeats := 4
	for i := 0; i < len(message)-maxRepeats; i++ {
		ch := message[i]
		count := 1
		for j := i + 1; j < len(message) && j < i+maxRepeats+3; j++ {
			if message[j] == ch {
				count++
			} else {
				break
			}
		}
		if count > maxRepeats {
			return fmt.Errorf("too many repeated characters")
		}
	}
	return nil
}

func defaultSpamKeywords() []string {
	return []string{
		"viagra", "cialis", "pharmacy", "pills", "casino", "gambling",
		"lottery", "jackpot", "bitcoin", "crypto", "forex", "seo",
		"backlinks", "ranking", "weight loss", "diet pills",
		"replica", "counterfeit", "adult", "porn", "xxx",
	}
}

func defaultBannedPhrases() []string {
	return []string{
		"click here", "free money", "make money fast",
		"work from home", "guaranteed income", "risk free",
		"act now", "congratulations you have won",
	}
}
