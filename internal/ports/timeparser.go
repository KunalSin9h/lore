package ports

import "time"

// TimeParserPort parses natural language time expressions into concrete times.
// Implemented by: adapters/timeparser.WhenParser
type TimeParserPort interface {
	Parse(expr string, from time.Time) (*time.Time, error)
}
