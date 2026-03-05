package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/justblue/samsa/internal/feature/comment"
	"github.com/justblue/samsa/pkg/queryparam"
)

func main() {
	f := &comment.CommentFilter{
		PaginationParams: queryparam.PaginationParams{
			Page:    500,
			Limit:   1,
			OrderBy: []string{"created_at:desc"},
		},
		ID: []uuid.UUID{
			uuid.New(), uuid.New(),
		},
		EntityType: "post",
		EntityID:   []uuid.UUID{uuid.New(), uuid.New()},
		IsPinned:   new(bool),
		IsReported: new(bool),
		IsArchived: new(bool),
		IsResolved: new(bool),
		IsDeleted:  new(bool),
	}

	query, countQuery, args := buildQuery(f)
	fmt.Println(query)
	fmt.Println()
	fmt.Println(countQuery)
	fmt.Println()
	fmt.Println(args)
}

func buildQuery(f *comment.CommentFilter) (string, string, []any) {
	orderByValue := f.ToSQL()
	if orderByValue == "" {
		orderByValue = "created_at DESC"
	}

	args := []any{false}
	argIndex := 2

	query := `SELECT * FROM comment WHERE is_deleted = $1`

	entityType := sql.NullString{String: f.EntityType, Valid: f.EntityType != ""}
	args = append(args, entityType)
	argIndex++

	if len(f.EntityID) > 0 {
		entityIDs := make([]string, len(f.EntityID))
		for i, id := range f.EntityID {
			entityIDs[i] = id.String()
		}
		query += fmt.Sprintf(" AND entity_id IN (%s)", strings.Join(entityIDs, ","))
	}

	if len(f.ID) > 0 {
		ids := make([]string, len(f.ID))
		for i, id := range f.ID {
			ids[i] = id.String()
		}
		query += fmt.Sprintf(" AND id IN (%s)", strings.Join(ids, ","))
	}

	if f.IsPinned != nil {
		query += fmt.Sprintf(" AND is_pinned = $%d", argIndex)
		args = append(args, *f.IsPinned)
		argIndex++
	}

	if f.IsReported != nil {
		query += fmt.Sprintf(" AND is_reported = $%d", argIndex)
		args = append(args, *f.IsReported)
		argIndex++
	}

	if f.IsArchived != nil {
		query += fmt.Sprintf(" AND is_archived = $%d", argIndex)
		args = append(args, *f.IsArchived)
		argIndex++
	}

	if f.IsResolved != nil {
		query += fmt.Sprintf(" AND is_resolved = $%d", argIndex)
		args = append(args, *f.IsResolved)
		argIndex++
	}

	countQuery := strings.Replace(query, "SELECT *", "SELECT COUNT(*)", 1)
	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderByValue, argIndex, argIndex+1)
	args = append(args, f.GetLimit(), f.GetOffset())

	return query, countQuery, args
}
