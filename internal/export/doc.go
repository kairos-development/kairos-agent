// Package export writes trade records to CSV with timezone-aware
// operator-local ISO 8601 timestamps.
//
// All monetary values are serialized from decimal.Decimal without
// precision loss. The CSV format is versioned once released publicly.
//
// This package must not import domain or service packages.
package export
