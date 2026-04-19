package db

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Helper functions for nullable types

func NullString(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func NullBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

func NullInt4(n int32) pgtype.Int4 {
	return pgtype.Int4{Int32: n, Valid: true}
}

func NullTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
