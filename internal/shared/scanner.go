package shared

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Scanner is the interface satisfied by both pgx.Row and pgx.Rows.
type Scanner interface {
	Scan(dest ...any) error
}

// ScanRow wraps pgxmock's Scan to handle **string destinations.
// pgxmock cannot scan a string value into a *string destination because
// the Go reflect system sees it as **string (pointer to *string).
// This function detects pointer-to-pointer types (like **string, **time.Time,
// **int, **float64) and uses sql.Null* intermediaries for scanning.
func ScanRow(rows Scanner, dest ...interface{}) error {
	type mappedDest struct {
		proxy interface{}
		fn    func()
	}

	mapped := make([]mappedDest, len(dest))

	for i, d := range dest {
		switch target := d.(type) {
		case **string:
			proxy := &sql.NullString{}
			mapped[i] = mappedDest{proxy: proxy, fn: func() {
				if proxy.Valid {
					s := proxy.String
					*target = &s
				} else {
					*target = nil
				}
			}}
		case **int:
			proxy := &sql.NullInt64{}
			mapped[i] = mappedDest{proxy: proxy, fn: func() {
				if proxy.Valid {
					v := int(proxy.Int64)
					*target = &v
				} else {
					*target = nil
				}
			}}
		case **float64:
			proxy := &sql.NullFloat64{}
			mapped[i] = mappedDest{proxy: proxy, fn: func() {
				if proxy.Valid {
					v := proxy.Float64
					*target = &v
				} else {
					*target = nil
				}
			}}
		case **bool:
			proxy := &sql.NullBool{}
			mapped[i] = mappedDest{proxy: proxy, fn: func() {
				if proxy.Valid {
					v := proxy.Bool
					*target = &v
				} else {
					*target = nil
				}
			}}
		case **time.Time:
			proxy := &pgtype.Timestamp{}
			mapped[i] = mappedDest{proxy: proxy, fn: func() {
				if proxy.Valid {
					t := proxy.Time
					*target = &t
				} else {
					*target = nil
				}
			}}
		default:
			_ = reflect.TypeOf(d)
			mapped[i] = mappedDest{proxy: d}
		}
	}

	proxies := make([]interface{}, len(mapped))
	for i, m := range mapped {
		proxies[i] = m.proxy
	}

	if err := rows.Scan(proxies...); err != nil {
		return fmt.Errorf("ScanRow: %w", err)
	}

	for _, m := range mapped {
		if m.fn != nil {
			m.fn()
		}
	}
	return nil
}
