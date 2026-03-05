package author

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

type CreateAuthorRequest struct {
	UserID                        uuid.UUID  `json:"user_id" validate:"required"`
	MediaID                       *uuid.UUID `json:"media_id"`
	StageName                     string     `json:"stage_name" validate:"required"`
	Gender                        string     `json:"gender"`
	Slug                          string     `json:"slug" validate:"required"`
	FirstName                     string     `json:"first_name"`
	LastName                      string     `json:"last_name"`
	DOB                           time.Time  `json:"dob"`
	Phone                         string     `json:"phone"`
	Bio                           string     `json:"bio"`
	Description                   string     `json:"description"`
	AcceptedTermsOfService        bool       `json:"accepted_terms_of_service"`
	EmailNewslettersAndChangelogs bool       `json:"email_newsletters_and_changelogs"`
	EmailPromotionsAndEvents      bool       `json:"email_promotions_and_events"`
}

type UpdateAuthorRequest struct {
	MediaID                       *uuid.UUID `json:"media_id"`
	StageName                     string     `json:"stage_name"`
	Gender                        string     `json:"gender"`
	FirstName                     string     `json:"first_name"`
	LastName                      string     `json:"last_name"`
	DOB                           *time.Time `json:"dob"`
	Phone                         string     `json:"phone"`
	Bio                           string     `json:"bio"`
	Description                   string     `json:"description"`
	EmailNewslettersAndChangelogs bool       `json:"email_newsletters_and_changelogs"`
	EmailPromotionsAndEvents      bool       `json:"email_promotions_and_events"`
}

type SetRecommendedRequest struct {
	IsRecommended bool `json:"is_recommended"`
}

type AuthorReposonse struct {
	sqlc.Author
}

type AuthorResponses struct {
	Authors []sqlc.Author             `json:"authors"`
	Meta    queryparam.PaginationMeta `json:"meta"`
}

type AuthorStat struct {
	TotalViews int64 `json:"total_views"`
}

var DefaultAuthorStat = AuthorStat{
	TotalViews: 0,
}
