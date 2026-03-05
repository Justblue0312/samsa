package author

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Author, error)
	GetBySlug(ctx context.Context, slug string) (*sqlc.Author, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*sqlc.Author, error)
	Create(ctx context.Context, user *sqlc.User, req *CreateAuthorRequest) (*sqlc.Author, error)
	Update(ctx context.Context, user *sqlc.User, authorID uuid.UUID, req *UpdateAuthorRequest) (*sqlc.Author, error)
	SoftDelete(ctx context.Context, user *sqlc.User, id uuid.UUID) error
	Restore(ctx context.Context, user *sqlc.User, id uuid.UUID) error
	Delete(ctx context.Context, user *sqlc.User, id uuid.UUID) error
	List(ctx context.Context, f *AuthorFilter) (*[]sqlc.Author, int64, error)
	SetRecommended(ctx context.Context, user *sqlc.User, authorID uuid.UUID, isRecommended bool) (*sqlc.Author, error)
}

type usecase struct {
	repo Repository
}

// NewUsecase returns a new AuthorUsecase backed by the given repository.
func NewUsecase(repo Repository) UseCase {
	return &usecase{repo: repo}
}

// SetRecommended implements [AuthorUsecase].
func (u *usecase) SetRecommended(ctx context.Context, user *sqlc.User, authorID uuid.UUID, isRecommended bool) (*sqlc.Author, error) {
	author, err := u.repo.GetByUserID(ctx, authorID)
	if err != nil {
		return nil, ErrAuthorNotFound
	}
	author.IsRecommended = &isRecommended
	return u.repo.Update(ctx, author)
}

// GetByID returns an author by ID. Returns ErrNotFound if missing.
func (u *usecase) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Author, error) {
	author, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return author, nil
}

// GetBySlug returns an author by slug. Returns ErrNotFound if missing.
func (u *usecase) GetBySlug(ctx context.Context, slug string) (*sqlc.Author, error) {
	author, err := u.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return author, nil
}

// GetByUserID returns an author by their user ID.
func (u *usecase) GetByUserID(ctx context.Context, userID uuid.UUID) (*sqlc.Author, error) {
	author, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return author, nil
}

// Create validates input and creates a new author.
func (u *usecase) Create(ctx context.Context, user *sqlc.User, req *CreateAuthorRequest) (*sqlc.Author, error) {
	if req.StageName == "" {
		return nil, ErrStageNameRequired
	}
	if req.Slug == "" {
		req.Slug = common.Slugify(req.StageName)
	}
	stat, _ := json.Marshal(DefaultAuthorStat)
	author := sqlc.Author{
		UserID:                        user.ID,
		MediaID:                       req.MediaID,
		StageName:                     req.StageName,
		Gender:                        req.Gender,
		Slug:                          req.Slug,
		FirstName:                     &req.FirstName,
		LastName:                      &req.LastName,
		DOB:                           &req.DOB,
		Phone:                         &req.Phone,
		Bio:                           &req.Bio,
		Description:                   &req.Description,
		AcceptedTermsOfService:        &req.AcceptedTermsOfService,
		EmailNewslettersAndChangelogs: &req.EmailNewslettersAndChangelogs,
		EmailPromotionsAndEvents:      &req.EmailPromotionsAndEvents,
		IsRecommended:                 new(bool),
		IsDeleted:                     false,
		Stats:                         stat,
		CreatedAt:                     common.Ptr(time.Now()),
		UpdatedAt:                     nil,
		DeletedAt:                     nil,
	}

	result, err := u.repo.Create(ctx, &author)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Update validates input and updates an existing author.
func (u *usecase) Update(ctx context.Context, user *sqlc.User, authorID uuid.UUID, req *UpdateAuthorRequest) (*sqlc.Author, error) {
	author, err := u.repo.GetByID(ctx, authorID)
	if err != nil {
		return nil, err
	}

	author.MediaID = req.MediaID
	author.StageName = req.StageName
	author.Gender = req.Gender
	author.FirstName = &req.FirstName
	author.LastName = &req.LastName
	author.DOB = req.DOB
	author.Phone = &req.Phone
	author.Bio = &req.Bio
	author.Description = &req.Description
	author.EmailNewslettersAndChangelogs = &req.EmailNewslettersAndChangelogs
	author.EmailPromotionsAndEvents = &req.EmailPromotionsAndEvents

	result, err := u.repo.Update(ctx, author)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// SoftDelete removes an author by ID.
func (u *usecase) SoftDelete(ctx context.Context, user *sqlc.User, authorID uuid.UUID) error {
	// Verify author exists before deleting.
	_, err := u.repo.GetByID(ctx, authorID)
	if err != nil {
		return err
	}

	return u.repo.SoftDelete(ctx, user.ID, authorID)
}

// List returns a filtered list of authors.
func (u *usecase) List(ctx context.Context, f *AuthorFilter) (*[]sqlc.Author, int64, error) {
	return u.repo.List(ctx, f)
}

// Delete implements [AuthorUsecase].
func (u *usecase) Delete(ctx context.Context, user *sqlc.User, authorID uuid.UUID) error {
	_, err := u.repo.GetByID(ctx, authorID)
	if err != nil {
		return err
	}

	return u.repo.Delete(ctx, user.ID, authorID)
}

// Restore implements [AuthorUsecase].
func (u *usecase) Restore(ctx context.Context, user *sqlc.User, authorID uuid.UUID) error {
	_, err := u.repo.GetByID(ctx, authorID)
	if err != nil {
		return err
	}

	return u.repo.Restore(ctx, user.ID, authorID)
}
