// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package utils

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Char struct {
	ID        int32
	Name      string
	Class     string
	CreatedAt int64
	UpdatedAt int64
	UserID    pgtype.Int4
}

type Debt struct {
	ID          int32
	Amount      int32
	Description string
	Date        pgtype.Date
	UserID      pgtype.Int4
}

type Player struct {
	ID        int32
	Name      string
	CreatedAt int64
	UpdatedAt int64
}
