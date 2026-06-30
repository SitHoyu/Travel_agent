package plan

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Create(ctx context.Context, record domain.PlanRecord) (domain.PlanRecord, error) {
	const query = `
INSERT INTO plans (
  user_id,
  request_id,
  session_id,
  status,
  title,
  destination,
  summary,
  final_answer,
  validation_summary,
  request_payload_json,
  plan_json,
  hotel_areas_json,
  executed_tools_json,
  tool_executions_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	result, err := r.db.ExecContext(
		ctx,
		query,
		record.UserID,
		nullIfEmpty(record.RequestID),
		nullIfEmpty(record.SessionID),
		record.Status,
		record.Title,
		record.Destination,
		nullIfEmpty(record.Summary),
		nullIfEmpty(record.FinalAnswer),
		nullIfEmpty(record.ValidationSummary),
		record.RequestPayloadJSON,
		record.PlanJSON,
		nullIfEmpty(record.HotelAreasJSON),
		nullIfEmpty(record.ExecutedToolsJSON),
		nullIfEmpty(record.ToolExecutionsJSON),
	)
	if err != nil {
		return domain.PlanRecord{}, fmt.Errorf("insert plan: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return domain.PlanRecord{}, fmt.Errorf("fetch inserted id: %w", err)
	}

	record.ID = id
	record.CreatedAt = time.Now()
	record.UpdatedAt = record.CreatedAt
	return record, nil
}

func (r *MySQLRepository) ListByUserID(ctx context.Context, userID int64, page, pageSize int) ([]domain.PlanRecord, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM plans WHERE user_id = ?", userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count plans: %w", err)
	}
	if total == 0 {
		return []domain.PlanRecord{}, 0, nil
	}

	const query = `
SELECT
  id,
  user_id,
  request_id,
  session_id,
  status,
  title,
  destination,
  summary,
  final_answer,
  validation_summary,
  request_payload_json,
  plan_json,
  hotel_areas_json,
  executed_tools_json,
  tool_executions_json,
  created_at,
  updated_at
FROM plans
WHERE user_id = ?
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?
`

	rows, err := r.db.QueryContext(ctx, query, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	records := make([]domain.PlanRecord, 0)
	for rows.Next() {
		record, err := scanPlanRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate plans: %w", err)
	}

	return records, total, nil
}

func (r *MySQLRepository) GetByIDAndUserID(ctx context.Context, id, userID int64) (domain.PlanRecord, bool, error) {
	const query = `
SELECT
  id,
  user_id,
  request_id,
  session_id,
  status,
  title,
  destination,
  summary,
  final_answer,
  validation_summary,
  request_payload_json,
  plan_json,
  hotel_areas_json,
  executed_tools_json,
  tool_executions_json,
  created_at,
  updated_at
FROM plans
WHERE id = ? AND user_id = ?
`

	row := r.db.QueryRowContext(ctx, query, id, userID)
	record, err := scanPlanRecord(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.PlanRecord{}, false, nil
		}
		return domain.PlanRecord{}, false, err
	}
	return record, true, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type planRowScanner interface {
	Scan(dest ...any) error
}

func scanPlanRecord(scanner planRowScanner) (domain.PlanRecord, error) {
	var record domain.PlanRecord
	var requestID sql.NullString
	var sessionID sql.NullString
	var summary sql.NullString
	var finalAnswer sql.NullString
	var validationSummary sql.NullString
	var requestJSON string
	var planJSON string
	var hotelAreasJSON sql.NullString
	var executedToolsJSON sql.NullString
	var toolExecutionsJSON sql.NullString

	err := scanner.Scan(
		&record.ID,
		&record.UserID,
		&requestID,
		&sessionID,
		&record.Status,
		&record.Title,
		&record.Destination,
		&summary,
		&finalAnswer,
		&validationSummary,
		&requestJSON,
		&planJSON,
		&hotelAreasJSON,
		&executedToolsJSON,
		&toolExecutionsJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return domain.PlanRecord{}, err
	}

	record.RequestID = requestID.String
	record.SessionID = sessionID.String
	record.Summary = summary.String
	record.FinalAnswer = finalAnswer.String
	record.ValidationSummary = validationSummary.String
	record.RequestPayloadJSON = requestJSON
	record.PlanJSON = planJSON
	record.HotelAreasJSON = hotelAreasJSON.String
	record.ExecutedToolsJSON = executedToolsJSON.String
	record.ToolExecutionsJSON = toolExecutionsJSON.String

	if err := decodePlanRecordJSON(&record); err != nil {
		return domain.PlanRecord{}, err
	}

	return record, nil
}

func decodePlanRecordJSON(record *domain.PlanRecord) error {
	if record.RequestPayloadJSON != "" {
		if err := json.Unmarshal([]byte(record.RequestPayloadJSON), &record.Request); err != nil {
			return fmt.Errorf("decode request payload: %w", err)
		}
	}
	if record.PlanJSON != "" {
		if err := json.Unmarshal([]byte(record.PlanJSON), &record.Plan); err != nil {
			return fmt.Errorf("decode plan json: %w", err)
		}
	}
	if record.HotelAreasJSON != "" {
		if err := json.Unmarshal([]byte(record.HotelAreasJSON), &record.HotelAreas); err != nil {
			return fmt.Errorf("decode hotel areas json: %w", err)
		}
	}
	if record.ExecutedToolsJSON != "" {
		if err := json.Unmarshal([]byte(record.ExecutedToolsJSON), &record.ExecutedTools); err != nil {
			return fmt.Errorf("decode executed tools json: %w", err)
		}
	}
	if record.ToolExecutionsJSON != "" {
		if err := json.Unmarshal([]byte(record.ToolExecutionsJSON), &record.ToolExecutions); err != nil {
			return fmt.Errorf("decode tool executions json: %w", err)
		}
	}
	return nil
}
