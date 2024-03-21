// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: queries.sql

package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const addJournalEntry = `-- name: AddJournalEntry :exec
INSERT INTO debt_journal (
    amount, description, user_id
) VALUES (
    $1, $2, $3
)
`

type AddJournalEntryParams struct {
	Amount      int64
	Description string
	UserID      pgtype.Int4
}

func (q *Queries) AddJournalEntry(ctx context.Context, arg AddJournalEntryParams) error {
	_, err := q.db.Exec(ctx, addJournalEntry, arg.Amount, arg.Description, arg.UserID)
	return err
}

const addPlayer = `-- name: AddPlayer :one
INSERT INTO player (
    name
) VALUES (
    lower($1)
) RETURNING id, name
`

func (q *Queries) AddPlayer(ctx context.Context, lower string) (Player, error) {
	row := q.db.QueryRow(ctx, addPlayer, lower)
	var i Player
	err := row.Scan(&i.ID, &i.Name)
	return i, err
}

const getAllDebts = `-- name: GetAllDebts :many
SELECT p.id, name, d.id, amount, last_updated, user_id FROM player p
JOIN debt d ON p.id = d.user_id
ORDER BY d.amount DESC, upper(p.name)
`

type GetAllDebtsRow struct {
	ID          int32
	Name        string
	ID_2        int32
	Amount      int64
	LastUpdated pgtype.Timestamp
	UserID      pgtype.Int4
}

func (q *Queries) GetAllDebts(ctx context.Context) ([]GetAllDebtsRow, error) {
	rows, err := q.db.Query(ctx, getAllDebts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetAllDebtsRow
	for rows.Next() {
		var i GetAllDebtsRow
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.ID_2,
			&i.Amount,
			&i.LastUpdated,
			&i.UserID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getBotSetup = `-- name: GetBotSetup :one
SELECT channel_id, message_id, created_at FROM bot_setup
WHERE created_at = (SELECT MAX(created_at) FROM bot_setup) LIMIT 1
`

func (q *Queries) GetBotSetup(ctx context.Context) (BotSetup, error) {
	row := q.db.QueryRow(ctx, getBotSetup)
	var i BotSetup
	err := row.Scan(&i.ChannelID, &i.MessageID, &i.CreatedAt)
	return i, err
}

const getDebt = `-- name: GetDebt :one
SELECT id, amount, last_updated, user_id FROM debt
WHERE user_id = $1 LIMIT 1
`

func (q *Queries) GetDebt(ctx context.Context, userID pgtype.Int4) (Debt, error) {
	row := q.db.QueryRow(ctx, getDebt, userID)
	var i Debt
	err := row.Scan(
		&i.ID,
		&i.Amount,
		&i.LastUpdated,
		&i.UserID,
	)
	return i, err
}

const getIdOfPlayer = `-- name: GetIdOfPlayer :one
SELECT id FROM player
WHERE name = lower($1) LIMIT 1
`

func (q *Queries) GetIdOfPlayer(ctx context.Context, lower string) (int32, error) {
	row := q.db.QueryRow(ctx, getIdOfPlayer, lower)
	var id int32
	err := row.Scan(&id)
	return id, err
}

const getJournalEntries = `-- name: GetJournalEntries :many
SELECT id, amount, description, date, user_id FROM debt_journal
WHERE user_id = $1
`

func (q *Queries) GetJournalEntries(ctx context.Context, userID pgtype.Int4) ([]DebtJournal, error) {
	rows, err := q.db.Query(ctx, getJournalEntries, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DebtJournal
	for rows.Next() {
		var i DebtJournal
		if err := rows.Scan(
			&i.ID,
			&i.Amount,
			&i.Description,
			&i.Date,
			&i.UserID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const numberOfPlayers = `-- name: NumberOfPlayers :one
SELECT COUNT(*) FROM player
`

func (q *Queries) NumberOfPlayers(ctx context.Context) (int64, error) {
	row := q.db.QueryRow(ctx, numberOfPlayers)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const putBotSetup = `-- name: PutBotSetup :one
INSERT INTO bot_setup (
    channel_id, message_id
) VALUES (
    $1, $2
) RETURNING channel_id, message_id, created_at
`

type PutBotSetupParams struct {
	ChannelID string
	MessageID string
}

func (q *Queries) PutBotSetup(ctx context.Context, arg PutBotSetupParams) (BotSetup, error) {
	row := q.db.QueryRow(ctx, putBotSetup, arg.ChannelID, arg.MessageID)
	var i BotSetup
	err := row.Scan(&i.ChannelID, &i.MessageID, &i.CreatedAt)
	return i, err
}

const setDebt = `-- name: SetDebt :exec
INSERT INTO debt (amount, user_id)
VALUES ($1, $2)
ON CONFLICT (user_id)
DO UPDATE SET amount = $1, last_updated = now()
WHERE debt.user_id = $2
`

type SetDebtParams struct {
	Amount int64
	UserID pgtype.Int4
}

func (q *Queries) SetDebt(ctx context.Context, arg SetDebtParams) error {
	_, err := q.db.Exec(ctx, setDebt, arg.Amount, arg.UserID)
	return err
}
