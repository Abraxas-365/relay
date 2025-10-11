package parser

import (
	"context"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type ParserRepository interface {
	Save(ctx context.Context, p Parser) error
	FindByID(ctx context.Context, id kernel.ParserID) (*Parser, error)
	FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*Parser, error)
}

// ParserEngine ejecuta parsers
type ParserEngine interface {
	Parse(ctx context.Context, parser Parser, message engine.Message, session *engine.Session) (*ParseResult, error)
}

// ParserSelector decide qué parser usar
type ParserSelector interface {
	SelectParser(ctx context.Context, message engine.Message, availableParsers []*Parser) (*Parser, error)
}
