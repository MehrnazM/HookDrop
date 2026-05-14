package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

)

type Drop struct {
	ID           string
	URLSlug      string
	SessionToken string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

type EventPreview struct {
	ID             string            `json:"id"`
	HTTPMethod     string            `json:"http_method"`
	Path           string            `json:"path"`
	ReceivedAt     time.Time         `json:"received_at"`
	IPAddress      string            `json:"ip_address"`
	HeadersPreview map[string]string `json:"headers_preview"`
	BodyPreview    string            `json:"body_preview"`
}

type EventDetail struct {
	ID             string          `json:"id"`
	HTTPMethod     string          `json:"http_method"`
	Path           string          `json:"path"`
	Headers        json.RawMessage `json:"headers"`
	QueryParams    json.RawMessage `json:"query_params"`
	Body           json.RawMessage `json:"body"`
	ReceivedAt     time.Time       `json:"received_at"`
	IPAddress      string          `json:"ip_address"`
	ResponseStatus *int            `json:"response_status"`
	ResponseBody   json.RawMessage `json:"response_body"`
}

type Queries struct {
	db *sql.DB
}

func NewQueries(db *sql.DB) *Queries {
	return &Queries{db: db}
}

func (q *Queries) InsertDrop(ctx context.Context, urlSlug, hashedToken string, expiresAt time.Time) error {
	_, err := q.db.ExecContext(ctx,
		`INSERT INTO drops (url_slug, session_token, expires_at) VALUES ($1, $2, $3)`,
		urlSlug, hashedToken, expiresAt,
	)
	return err
}

func (q *Queries) GetDropBySlug(ctx context.Context, slug string) (*Drop, error) {
	d := &Drop{}
	err := q.db.QueryRowContext(ctx,
		`SELECT id, url_slug, session_token, created_at, expires_at FROM drops WHERE url_slug = $1`,
		slug,
	).Scan(&d.ID, &d.URLSlug, &d.SessionToken, &d.CreatedAt, &d.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (q *Queries) CountEventsForDrop(ctx context.Context, dropID string) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM webhook_events WHERE drop_id = $1`, dropID,
	).Scan(&n)
	return n, err
}

func (q *Queries) ListEventsForDrop(ctx context.Context, dropID string, page, limit int) ([]EventPreview, int, error) {
	var total int
	if err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM webhook_events WHERE drop_id = $1`, dropID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := q.db.QueryContext(ctx, `
		SELECT id, http_method, path, received_at,
		       COALESCE(ip_address::text, ''),
		       headers,
		       COALESCE(LEFT(body::text, 100), '')
		FROM webhook_events
		WHERE drop_id = $1
		ORDER BY received_at DESC
		LIMIT $2 OFFSET $3`,
		dropID, limit, (page-1)*limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []EventPreview
	for rows.Next() {
		var e EventPreview
		var headersRaw []byte
		if err := rows.Scan(&e.ID, &e.HTTPMethod, &e.Path, &e.ReceivedAt, &e.IPAddress, &headersRaw, &e.BodyPreview); err != nil {
			return nil, 0, err
		}
		e.HeadersPreview = extractHeadersPreview(headersRaw)
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (q *Queries) GetEventDetail(ctx context.Context, dropID, eventID string) (*EventDetail, error) {
	e := &EventDetail{}
	var ipAddr sql.NullString
	var qpBytes, bodyBytes, respBodyBytes []byte
	var respStatus sql.NullInt32

	// drop_responses is LEFT JOIN'd here for future use; the table is written to
	// by the response-capture feature (not yet implemented).
	err := q.db.QueryRowContext(ctx, `
		SELECT we.id, we.http_method, we.path,
		       we.headers, we.query_params, we.body,
		       we.received_at, we.ip_address,
		       dr.status_code, dr.response_body
		FROM webhook_events we
		LEFT JOIN drop_responses dr ON dr.webhook_event_id = we.id
		WHERE we.id = $1 AND we.drop_id = $2`,
		eventID, dropID,
	).Scan(
		&e.ID, &e.HTTPMethod, &e.Path,
		&e.Headers, &qpBytes, &bodyBytes,
		&e.ReceivedAt, &ipAddr,
		&respStatus, &respBodyBytes,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if ipAddr.Valid {
		e.IPAddress = ipAddr.String
	}
	e.QueryParams = json.RawMessage(qpBytes)
	e.Body = json.RawMessage(bodyBytes)
	if respStatus.Valid {
		s := int(respStatus.Int32)
		e.ResponseStatus = &s
	}
	e.ResponseBody = json.RawMessage(respBodyBytes)
	return e, nil
}

func (q *Queries) DeleteDrop(ctx context.Context, dropID string) error {
	res, err := q.db.ExecContext(ctx, `DELETE FROM drops WHERE id = $1`, dropID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func extractHeadersPreview(raw []byte) map[string]string {
	var all map[string]string
	if err := json.Unmarshal(raw, &all); err != nil {
		return map[string]string{}
	}
	preview := make(map[string]string, 3)
	if v, ok := all["Content-Type"]; ok {
		preview["Content-Type"] = v
	}
	for k, v := range all {
		if len(preview) >= 3 {
			break
		}
		if k == "Content-Type" {
			continue
		}
		preview[k] = v
	}
	return preview
}
