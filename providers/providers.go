package providers

import (
	"fmt"
	"os"

	http "github.com/bogdanfinn/fhttp"
)

// Removed unnecessary imports for providers other than OpenAI

func GetMainText(line string, provider string, input string) string {
	// Since we only support OpenAI, return an error for other providers
	if provider != "openai" {
		return fmt.Errorf("Unsupported provider: %s", provider)
	}
	return openai.GetMainText(line)
}

func NewRequest(input string, params structs.Params, extraOptions structs.ExtraOptions) (*http.Response, error) {
	// Validate provider (same logic as before)
	validProvider := false
	for _, str := range []string{"", "openai"} { // Shortened list
		if str == params.Provider {
			validProvider = true
			break
		}
	}
	if !validProvider {
		fmt.Fprintln(os.Stderr, "Invalid provider")
		os.Exit(1)
	}

	// Handle OpenAI request as before
	if params.Provider == "openai" {
		return openai.NewRequest(input, params)
	}

	// No need for a default case since OpenAI is the only supported provider now

}