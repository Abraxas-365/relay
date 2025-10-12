package parserinfra

import (
	"context"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/parser"
	"regexp"
)

type RegexParserEngine struct{}

func NewRegexParserEngine() *RegexParserEngine {
	return &RegexParserEngine{}
}

func (rpe *RegexParserEngine) Parse(ctx context.Context, p parser.Parser, msg engine.Message, session *engine.Session) (*parser.ParseResult, error) {
	messageText := msg.Content.Text

	// Iterar patterns
	for _, pattern := range p.Config.Patterns {
		re, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue
		}

		if re.MatchString(messageText) {
			// Match encontrado, ejecutar acciones
			return &parser.ParseResult{
				Success:       true,
				Actions:       pattern.Actions,
				ShouldRespond: hasResponseAction(pattern.Actions),
				Response:      extractResponse(pattern.Actions),
			}, nil
		}
	}

	// No match
	return &parser.ParseResult{
		Success: false,
	}, nil
}

func hasResponseAction(actions []parser.Action) bool {
	for _, action := range actions {
		if action.Type == "response" {
			return true
		}
	}
	return false
}

func extractResponse(actions []parser.Action) string {
	for _, action := range actions {
		if action.Type == "response" {
			if msg, ok := action.Config["message"].(string); ok {
				return msg
			}
		}
	}
	return ""
}
