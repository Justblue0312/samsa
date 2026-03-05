package session

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks
import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/gen/sqlc"
)

var (
	ErrNotFound = errors.New("session not found")
)

type Repository interface {
	Create(ctx context.Context, session *sqlc.Session) (*sqlc.Session, error)
	GetByToken(ctx context.Context, token string, isExpired bool) (*sqlc.Session, *sqlc.User, error)
	DeleteExpire(ctx context.Context) error
	DeleteByUserId(ctx context.Context, userId uuid.UUID) error
}

type repository struct {
	q *sqlc.Queries
}

// DeleteByUserId implements [Repository].
func (r *repository) DeleteByUserId(ctx context.Context, userId uuid.UUID) error {
	return r.q.DeleteSessionsByUserId(ctx, userId)
}

func NewRepository(q *sqlc.Queries) Repository {
	return &repository{q: q}
}

func (r *repository) Create(ctx context.Context, session *sqlc.Session) (*sqlc.Session, error) {
	s, err := r.q.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:     session.UserID,
		Token:      session.Token,
		IPAddress:  session.IPAddress,
		UserAgent:  session.UserAgent,
		DeviceInfo: session.DeviceInfo,
		Scopes:     session.Scopes,
		Metadata:   session.Metadata,
		IsActive:   session.IsActive,
		ExpiresAt:  session.ExpiresAt,
		CreatedAt:  session.CreatedAt,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *repository) DeleteExpire(ctx context.Context) error {
	return r.q.DeleteExpiredSessions(ctx)
}

func (r *repository) GetByToken(ctx context.Context, token string, isExpired bool) (*sqlc.Session, *sqlc.User, error) {
	row, err := r.q.GetSessionByToken(ctx, sqlc.GetSessionByTokenParams{
		Token:     token,
		IsExpired: isExpired,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	return &row.Session, &row.User, nil
}
