package operation_setting

import "strings"

var DemoSiteEnabled = false
var SelfUseModeEnabled = false

var AutomaticDisableKeywords = []string{
	"Your credit balance is too low",
	"This organization has been disabled.",
	"You exceeded your current quota",
	"Permission denied",
	"The security token included in the request is invalid",
	"Operation not allowed",
	"Your account is not authorized",
}

func AutomaticDisableKeywordsToString() string {
	return strings.Join(AutomaticDisableKeywords, "\n")
}

func AutomaticDisableKeywordsFromString(s string) {
	AutomaticDisableKeywords = []string{}
	ak := strings.SplitSeq(s, "\n")
	for k := range ak {
		k = strings.TrimSpace(k)
		k = strings.ToLower(k)
		if k != "" {
			AutomaticDisableKeywords = append(AutomaticDisableKeywords, k)
		}
	}
}

// AutomaticRetryKeywords lets an upstream error whose status is outside the
// retry rules move on to the next channel when its message contains one of
// these phrases (case insensitive). Empty by default.
var AutomaticRetryKeywords = []string{}

func AutomaticRetryKeywordsToString() string {
	return strings.Join(AutomaticRetryKeywords, "\n")
}

func AutomaticRetryKeywordsFromString(s string) {
	AutomaticRetryKeywords = []string{}
	for k := range strings.SplitSeq(s, "\n") {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" {
			AutomaticRetryKeywords = append(AutomaticRetryKeywords, k)
		}
	}
}
