package parserengines

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/parser"
)

type RegexParserEngine struct{}

func NewRegexParserEngine() *RegexParserEngine {
	return &RegexParserEngine{}
}

func (rpe *RegexParserEngine) Parse(ctx context.Context, p parser.Parser, msg engine.Message, session *engine.Session) (*parser.ParseResult, error) {
	// Get typed config
	config, err := p.GetConfigStruct()
	if err != nil {
		return parser.NewFailureResult(p.ID, p.Name, err), err
	}

	regexConfig, ok := config.(parser.RegexParserConfig)
	if !ok {
		err := fmt.Errorf("invalid config type for regex parser")
		return parser.NewFailureResult(p.ID, p.Name, err), err
	}

	messageText := msg.Content.Text
	result := parser.NewParseResult(p.ID, p.Name)

	// Iterate through patterns
	for _, pattern := range regexConfig.Patterns {
		re, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			// Log error but continue with next pattern
			continue
		}

		if matches := re.FindStringSubmatch(messageText); matches != nil {
			// Match found
			result.Success = true
			result.Actions = pattern.Actions
			result.ShouldRespond = hasResponseAction(pattern.Actions)
			result.Response = extractResponse(pattern.Actions)
			result.Confidence = 1.0 // Regex matches are binary

			// Extract capture groups if defined
			if len(pattern.CaptureGroups) > 0 && len(matches) > 1 {
				for name, index := range pattern.CaptureGroups {
					if index < len(matches) {
						result.SetExtractedValue(name, matches[index])
					}
				}
			}

			// Store pattern info in metadata
			result.Metadata["matched_pattern"] = pattern.Name
			result.Metadata["pattern"] = pattern.Pattern

			return result, nil
		}
	}

	// No match found
	result.Success = false
	result.Confidence = 0.0
	return result, nil
}

func (rpe *RegexParserEngine) SupportsType(parserType parser.ParserType) bool {
	return parserType == parser.ParserTypeRegex
}

func (rpe *RegexParserEngine) ValidateConfig(config parser.ParserConfig) error {
	regexConfig, ok := config.(parser.RegexParserConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected RegexParserConfig")
	}

	// Validate the config using its built-in method
	if err := regexConfig.Validate(); err != nil {
		return err
	}

	// Additional validation: check if patterns compile
	for i, pattern := range regexConfig.Patterns {
		if _, err := regexp.Compile(pattern.Pattern); err != nil {
			return fmt.Errorf("pattern %d (%s) failed to compile: %w", i, pattern.Name, err)
		}
	}

	return nil
}

// Helper functions

func hasResponseAction(actions []parser.Action) bool {
	for _, action := range actions {
		if action.Type == parser.ActionTypeResponse {
			return true
		}
	}
	return false
}

func extractResponse(actions []parser.Action) string {
	for _, action := range actions {
		if action.Type == parser.ActionTypeResponse {
			if msg, ok := action.Config["message"].(string); ok {
				return msg
			}
		}
	}
	return ""
}

