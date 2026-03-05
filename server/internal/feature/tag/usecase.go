package tag

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	// Tag CRUD
	CreateTag(ctx context.Context, ownerID uuid.UUID, req *CreateTagRequest) (*TagResponse, error)
	GetTag(ctx context.Context, id uuid.UUID, entityType sqlc.EntityType) (*TagResponse, error)
	UpdateTag(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateTagRequest, entityType sqlc.EntityType) (*TagResponse, error)
	DeleteTag(ctx context.Context, id uuid.UUID, userID uuid.UUID, entityType sqlc.EntityType) error

	// Tag queries
	GetTagsByEntity(ctx context.Context, entityID uuid.UUID, entityType sqlc.EntityType, isHidden, isSystem, isRecommended *bool) ([]TagResponse, error)
	GetTagsByOwner(ctx context.Context, ownerID uuid.UUID, entityType sqlc.EntityType, isHidden, isSystem, isRecommended *bool, limit, offset int32) ([]TagResponse, int64, error)
	GetTagsByIDs(ctx context.Context, tagIDs []uuid.UUID, entityType sqlc.EntityType) ([]TagResponse, error)

	// Search and list
	SearchTags(ctx context.Context, entityType sqlc.EntityType, searchQuery *string, isHidden, isSystem, isRecommended *bool, limit, offset int32) ([]TagResponse, int64, error)
	ListTags(ctx context.Context, f *TagFilter, entityType sqlc.EntityType) ([]TagResponse, int64, error)

	// Utility
	GetEntityIDsByTagNames(ctx context.Context, tagNames []string, entityType sqlc.EntityType) ([]uuid.UUID, error)
	CountTagsByEntity(ctx context.Context, entityID uuid.UUID, entityType sqlc.EntityType) (int64, error)
	CountTagsByOwner(ctx context.Context, ownerID uuid.UUID, entityType sqlc.EntityType) (int64, error)
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

// CreateTag creates a new tag
func (uc *usecase) CreateTag(ctx context.Context, ownerID uuid.UUID, req *CreateTagRequest) (*TagResponse, error) {
	now := time.Now()

	tag := &sqlc.Tag{
		ID:            uuid.New(),
		OwnerID:       ownerID,
		Name:          req.Name,
		Description:   req.Description,
		Color:         req.Color,
		EntityType:    req.EntityType,
		EntityID:      *req.EntityID,
		IsHidden:      req.IsHidden,
		IsSystem:      req.IsSystem,
		IsRecommended: req.IsRecommended,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	created, err := uc.repo.Create(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("usecase.CreateTag: %w", err)
	}

	return ToTagResponse(created), nil
}

// GetTag retrieves a tag by ID
func (uc *usecase) GetTag(ctx context.Context, id uuid.UUID, entityType sqlc.EntityType) (*TagResponse, error) {
	tag, err := uc.repo.GetByID(ctx, id, entityType)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetTag: %w", err)
	}
	return ToTagResponse(tag), nil
}

// UpdateTag updates a tag
func (uc *usecase) UpdateTag(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateTagRequest, entityType sqlc.EntityType) (*TagResponse, error) {
	tag, err := uc.repo.GetByID(ctx, id, entityType)
	if err != nil {
		return nil, fmt.Errorf("usecase.UpdateTag: %w", err)
	}

	// Check ownership
	if tag.OwnerID != userID {
		return nil, ErrNotOwner
	}

	now := time.Now()

	if req.Name != nil {
		tag.Name = *req.Name
	}
	if req.Description != nil {
		tag.Description = req.Description
	}
	if req.Color != nil {
		tag.Color = *req.Color
	}
	if req.IsHidden != nil {
		tag.IsHidden = req.IsHidden
	}
	if req.IsSystem != nil {
		tag.IsSystem = req.IsSystem
	}
	if req.IsRecommended != nil {
		tag.IsRecommended = req.IsRecommended
	}
	tag.UpdatedAt = &now

	updated, err := uc.repo.Update(ctx, tag, entityType)
	if err != nil {
		return nil, fmt.Errorf("usecase.UpdateTag: %w", err)
	}

	return ToTagResponse(updated), nil
}

// DeleteTag deletes a tag
func (uc *usecase) DeleteTag(ctx context.Context, id uuid.UUID, userID uuid.UUID, entityType sqlc.EntityType) error {
	tag, err := uc.repo.GetByID(ctx, id, entityType)
	if err != nil {
		return fmt.Errorf("usecase.DeleteTag: %w", err)
	}

	// Check ownership
	if tag.OwnerID != userID {
		return ErrNotOwner
	}

	if err := uc.repo.Delete(ctx, id, entityType); err != nil {
		return fmt.Errorf("usecase.DeleteTag: %w", err)
	}

	return nil
}

// GetTagsByEntity retrieves tags for an entity
func (uc *usecase) GetTagsByEntity(ctx context.Context, entityID uuid.UUID, entityType sqlc.EntityType, isHidden, isSystem, isRecommended *bool) ([]TagResponse, error) {
	tags, err := uc.repo.GetTagsByEntityID(ctx, entityID, entityType, isHidden, isSystem, isRecommended)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetTagsByEntity: %w", err)
	}

	result := make([]TagResponse, len(tags))
	for i, t := range tags {
		result[i] = *ToTagResponse(t)
	}

	return result, nil
}

// GetTagsByOwner retrieves tags by owner
func (uc *usecase) GetTagsByOwner(ctx context.Context, ownerID uuid.UUID, entityType sqlc.EntityType, isHidden, isSystem, isRecommended *bool, limit, offset int32) ([]TagResponse, int64, error) {
	tags, err := uc.repo.GetTagsByOwnerID(ctx, ownerID, entityType, isHidden, isSystem, isRecommended, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.GetTagsByOwner: %w", err)
	}

	count, err := uc.repo.CountTagsByOwner(ctx, ownerID, entityType)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.GetTagsByOwner count: %w", err)
	}

	result := make([]TagResponse, len(tags))
	for i, t := range tags {
		result[i] = *ToTagResponse(t)
	}

	return result, int64(count), nil
}

// GetTagsByIDs retrieves tags by IDs
func (uc *usecase) GetTagsByIDs(ctx context.Context, tagIDs []uuid.UUID, entityType sqlc.EntityType) ([]TagResponse, error) {
	if len(tagIDs) == 0 {
		return []TagResponse{}, nil
	}

	tags, err := uc.repo.GetTagsByIDs(ctx, tagIDs, entityType)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetTagsByIDs: %w", err)
	}

	result := make([]TagResponse, len(tags))
	for i, t := range tags {
		result[i] = *ToTagResponse(t)
	}

	return result, nil
}

// SearchTags searches for tags
func (uc *usecase) SearchTags(ctx context.Context, entityType sqlc.EntityType, searchQuery *string, isHidden, isSystem, isRecommended *bool, limit, offset int32) ([]TagResponse, int64, error) {
	tags, err := uc.repo.SearchTags(ctx, entityType, searchQuery, isHidden, isSystem, isRecommended, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.SearchTags: %w", err)
	}

	// Get total count
	count := int64(len(tags))
	if len(tags) == int(limit) {
		// If we got full page, there might be more
		count = int64(limit + offset + 1) // Approximate
	}

	result := make([]TagResponse, len(tags))
	for i, t := range tags {
		result[i] = *ToTagResponse(t)
	}

	return result, count, nil
}

// ListTags lists tags with filters
func (uc *usecase) ListTags(ctx context.Context, f *TagFilter, entityType sqlc.EntityType) ([]TagResponse, int64, error) {
	tags, total, err := uc.repo.ListTags(ctx, f, entityType)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.ListTags: %w", err)
	}

	result := make([]TagResponse, len(tags))
	for i, t := range tags {
		result[i] = *ToTagResponse(t)
	}

	return result, total, nil
}

// GetEntityIDsByTagNames retrieves entity IDs by tag names
func (uc *usecase) GetEntityIDsByTagNames(ctx context.Context, tagNames []string, entityType sqlc.EntityType) ([]uuid.UUID, error) {
	if len(tagNames) == 0 {
		return []uuid.UUID{}, nil
	}

	entityIDs, err := uc.repo.GetEntityIDsByTagNames(ctx, tagNames, entityType)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetEntityIDsByTagNames: %w", err)
	}

	return entityIDs, nil
}

// CountTagsByEntity counts tags for an entity
func (uc *usecase) CountTagsByEntity(ctx context.Context, entityID uuid.UUID, entityType sqlc.EntityType) (int64, error) {
	count, err := uc.repo.CountTagsByEntity(ctx, entityID, entityType)
	if err != nil {
		return 0, fmt.Errorf("usecase.CountTagsByEntity: %w", err)
	}
	return int64(count), nil
}

// CountTagsByOwner counts tags by owner
func (uc *usecase) CountTagsByOwner(ctx context.Context, ownerID uuid.UUID, entityType sqlc.EntityType) (int64, error) {
	count, err := uc.repo.CountTagsByOwner(ctx, ownerID, entityType)
	if err != nil {
		return 0, fmt.Errorf("usecase.CountTagsByOwner: %w", err)
	}
	return int64(count), nil
}

// Helper functions

// ToTagResponse converts sqlc.Tag to TagResponse
func ToTagResponse(t *sqlc.Tag) *TagResponse {
	if t == nil {
		return nil
	}

	resp := &TagResponse{
		ID:          t.ID,
		OwnerID:     t.OwnerID,
		Name:        t.Name,
		Description: t.Description,
		Color:       t.Color,
		EntityType:  t.EntityType,
		EntityID:    &t.EntityID,
	}

	if t.IsHidden != nil {
		resp.IsHidden = *t.IsHidden
	}
	if t.IsSystem != nil {
		resp.IsSystem = *t.IsSystem
	}
	if t.IsRecommended != nil {
		resp.IsRecommended = *t.IsRecommended
	}
	if t.CreatedAt != nil {
		resp.CreatedAt = *t.CreatedAt
	}
	if t.UpdatedAt != nil {
		resp.UpdatedAt = *t.UpdatedAt
	}

	return resp
}

// TagResponse represents a tag in API responses
type TagResponse struct {
	ID            uuid.UUID       `json:"id"`
	OwnerID       uuid.UUID       `json:"owner_id"`
	Name          string          `json:"name"`
	Description   *string         `json:"description,omitempty"`
	Color         string          `json:"color"`
	EntityType    sqlc.EntityType `json:"entity_type"`
	EntityID      *uuid.UUID      `json:"entity_id,omitempty"`
	IsHidden      bool            `json:"is_hidden"`
	IsSystem      bool            `json:"is_system"`
	IsRecommended bool            `json:"is_recommended"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// CreateTagRequest represents a request to create a tag
type CreateTagRequest struct {
	Name          string          `json:"name" validate:"required,max=50"`
	Description   *string         `json:"description" validate:"omitempty,max=255"`
	Color         string          `json:"color" validate:"required,hexcolor"`
	EntityType    sqlc.EntityType `json:"entity_type" validate:"required,oneof=story chapter comment submission"`
	EntityID      *uuid.UUID      `json:"entity_id" validate:"omitempty,uuid"`
	IsHidden      *bool           `json:"is_hidden"`
	IsSystem      *bool           `json:"is_system"`
	IsRecommended *bool           `json:"is_recommended"`
}

// UpdateTagRequest represents a request to update a tag
type UpdateTagRequest struct {
	Name          *string `json:"name" validate:"omitempty,max=50"`
	Description   *string `json:"description" validate:"omitempty,max=255"`
	Color         *string `json:"color" validate:"omitempty,hexcolor"`
	IsHidden      *bool   `json:"is_hidden"`
	IsSystem      *bool   `json:"is_system"`
	IsRecommended *bool   `json:"is_recommended"`
}

// TagListResponse represents a paginated list of tags
type TagListResponse struct {
	Tags  []TagResponse `json:"tags"`
	Total int64         `json:"total"`
	Page  int32         `json:"page"`
	Limit int32         `json:"limit"`
}
