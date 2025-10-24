package telemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SQLiteExporter exports spans to SQLite database
type SQLiteExporter struct {
	db *sql.DB
}

// NewSQLiteExporter creates a new SQLite exporter
func NewSQLiteExporter(dbPath string) (*SQLiteExporter, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	exporter := &SQLiteExporter{db: db}

	if err := exporter.initSchema(); err != nil {
		return nil, err
	}

	return exporter, nil
}

// initSchema creates the database schema
func (e *SQLiteExporter) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS spans (
		trace_id TEXT NOT NULL,
		span_id TEXT PRIMARY KEY,
		parent_span_id TEXT,
		name TEXT NOT NULL,
		kind TEXT NOT NULL,
		start_time INTEGER NOT NULL,
		end_time INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		status_code TEXT NOT NULL,
		status_message TEXT,
		attributes TEXT,
		events TEXT,
		links TEXT,
		resource TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_trace_id ON spans(trace_id);
	CREATE INDEX IF NOT EXISTS idx_parent_span_id ON spans(parent_span_id);
	CREATE INDEX IF NOT EXISTS idx_start_time ON spans(start_time);
	CREATE INDEX IF NOT EXISTS idx_name ON spans(name);

	CREATE VIRTUAL TABLE IF NOT EXISTS spans_fts USING fts5(
		span_id UNINDEXED,
		name,
		attributes,
		content='spans',
		content_rowid='rowid'
	);
	`

	_, err := e.db.Exec(schema)
	return err
}

// ExportSpans exports a batch of spans
func (e *SQLiteExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO spans
		(trace_id, span_id, parent_span_id, name, kind, start_time, end_time, duration_ms,
		 status_code, status_message, attributes, events, links, resource)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, span := range spans {
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		var parentSpanID *string
		if span.Parent().SpanID().IsValid() {
			pid := span.Parent().SpanID().String()
			parentSpanID = &pid
		}

		// Serialize attributes
		attrs := make(map[string]interface{})
		for _, attr := range span.Attributes() {
			attrs[string(attr.Key)] = attr.Value.AsInterface()
		}
		attrsJSON, _ := json.Marshal(attrs)

		// Serialize events
		events := make([]map[string]interface{}, len(span.Events()))
		for i, event := range span.Events() {
			eventAttrs := make(map[string]interface{})
			for _, attr := range event.Attributes {
				eventAttrs[string(attr.Key)] = attr.Value.AsInterface()
			}
			events[i] = map[string]interface{}{
				"name":       event.Name,
				"timestamp":  event.Time.UnixNano(),
				"attributes": eventAttrs,
			}
		}
		eventsJSON, _ := json.Marshal(events)

		// Serialize links
		links := make([]map[string]interface{}, len(span.Links()))
		for i, link := range span.Links() {
			linkAttrs := make(map[string]interface{})
			for _, attr := range link.Attributes {
				linkAttrs[string(attr.Key)] = attr.Value.AsInterface()
			}
			links[i] = map[string]interface{}{
				"trace_id":   link.SpanContext.TraceID().String(),
				"span_id":    link.SpanContext.SpanID().String(),
				"attributes": linkAttrs,
			}
		}
		linksJSON, _ := json.Marshal(links)

		// Serialize resource
		resourceJSON, _ := json.Marshal(span.Resource().Attributes())

		// Calculate duration
		duration := span.EndTime().Sub(span.StartTime()).Milliseconds()

		_, err = stmt.ExecContext(ctx,
			traceID,
			spanID,
			parentSpanID,
			span.Name(),
			span.SpanKind().String(),
			span.StartTime().UnixNano(),
			span.EndTime().UnixNano(),
			duration,
			span.Status().Code.String(),
			span.Status().Description,
			string(attrsJSON),
			string(eventsJSON),
			string(linksJSON),
			string(resourceJSON),
		)

		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Shutdown closes the database connection
func (e *SQLiteExporter) Shutdown(ctx context.Context) error {
	return e.db.Close()
}

// Close is an alias for Shutdown
func (e *SQLiteExporter) Close() error {
	return e.db.Close()
}

// QuerySpans queries spans by trace ID
func (e *SQLiteExporter) QuerySpans(traceID string) ([]SpanData, error) {
	rows, err := e.db.Query(`
		SELECT trace_id, span_id, parent_span_id, name, kind, start_time, end_time,
		       duration_ms, status_code, status_message, attributes, events
		FROM spans
		WHERE trace_id = ?
		ORDER BY start_time
	`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spans := []SpanData{}
	for rows.Next() {
		var span SpanData
		var parentSpanID sql.NullString
		var statusMessage sql.NullString
		var attrsJSON, eventsJSON string

		err := rows.Scan(
			&span.TraceID,
			&span.SpanID,
			&parentSpanID,
			&span.Name,
			&span.Kind,
			&span.StartTime,
			&span.EndTime,
			&span.DurationMs,
			&span.StatusCode,
			&statusMessage,
			&attrsJSON,
			&eventsJSON,
		)
		if err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if statusMessage.Valid {
			span.StatusMessage = statusMessage.String
		}

		json.Unmarshal([]byte(attrsJSON), &span.Attributes)
		json.Unmarshal([]byte(eventsJSON), &span.Events)

		spans = append(spans, span)
	}

	return spans, rows.Err()
}

// ListRecentTraces lists recent traces
func (e *SQLiteExporter) ListRecentTraces(limit int) ([]TraceInfo, error) {
	rows, err := e.db.Query(`
		SELECT trace_id, MIN(start_time) as start_time, MAX(end_time) as end_time,
		       COUNT(*) as span_count
		FROM spans
		GROUP BY trace_id
		ORDER BY start_time DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	traces := []TraceInfo{}
	for rows.Next() {
		var trace TraceInfo
		err := rows.Scan(&trace.TraceID, &trace.StartTime, &trace.EndTime, &trace.SpanCount)
		if err != nil {
			return nil, err
		}

		traces = append(traces, trace)
	}

	return traces, rows.Err()
}

// SpanData represents a span from the database
type SpanData struct {
	TraceID       string
	SpanID        string
	ParentSpanID  string
	Name          string
	Kind          string
	StartTime     int64
	EndTime       int64
	DurationMs    int64
	StatusCode    string
	StatusMessage string
	Attributes    map[string]interface{}
	Events        []map[string]interface{}
}

// TraceInfo represents trace metadata
type TraceInfo struct {
	TraceID   string
	StartTime int64
	EndTime   int64
	SpanCount int
}

// FormatSpan formats a span for display
func FormatSpan(span SpanData) string {
	startTime := time.Unix(0, span.StartTime)
	return fmt.Sprintf("[%s] %s (%dms) - %s",
		startTime.Format("15:04:05.000"),
		span.Name,
		span.DurationMs,
		span.StatusCode,
	)
}
