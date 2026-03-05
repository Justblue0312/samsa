package deps

import (
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/chapter"
	"github.com/justblue/samsa/internal/feature/document"
	document_folder "github.com/justblue/samsa/internal/feature/document_folder"
	"github.com/justblue/samsa/internal/feature/flag"
	"github.com/justblue/samsa/internal/feature/genre"
	"github.com/justblue/samsa/internal/feature/notification"
	notificationrecipient "github.com/justblue/samsa/internal/feature/notification_recipient"
	oauthaccount "github.com/justblue/samsa/internal/feature/oauth_account"
	"github.com/justblue/samsa/internal/feature/session"
	"github.com/justblue/samsa/internal/feature/spinnet"
	"github.com/justblue/samsa/internal/feature/story"
	"github.com/justblue/samsa/internal/feature/submission"
	submission_assignment "github.com/justblue/samsa/internal/feature/submission_assignment"
	submission_status_history "github.com/justblue/samsa/internal/feature/submission_status_history"
	"github.com/justblue/samsa/internal/feature/user"
)

type Repositories struct {
	User                    user.Repository
	Session                 session.Repository
	OAuthAccount            oauthaccount.Repository
	Notification            notification.Repository
	NotificationRecipient   notificationrecipient.Repository
	Submission              submission.Repository
	SubmissionAssignment    submission_assignment.Repository
	SubmissionStatusHistory submission_status_history.Repository
	Flag                    flag.Repository
	Spinnet                 spinnet.Repository
	Story                   story.Repository
	Genre                   genre.Repository
	Chapter                 chapter.Repository
	Document                document.Repository
	DocumentFolder          document_folder.Repository
}

func NewRepositories(db sqlc.DBTX, cfg *config.Config) *Repositories {
	q := sqlc.New(db)
	return &Repositories{
		User:                    user.NewRepository(q, cfg, nil),
		OAuthAccount:            oauthaccount.NewRepository(q, cfg),
		Session:                 session.NewRepository(q),
		Notification:            notification.NewRepository(q, nil, cfg),
		NotificationRecipient:   notificationrecipient.NewRepository(q),
		Submission:              submission.NewRepository(nil, q, cfg, nil),
		SubmissionAssignment:    submission_assignment.NewRepository(nil, q, cfg, nil),
		SubmissionStatusHistory: submission_status_history.NewRepository(nil, q, cfg, nil),
		Flag:                    flag.NewRepository(q),
		Spinnet:                 spinnet.NewRepository(q),
		Story:                   story.NewRepository(db),
		Genre:                   genre.NewRepository(db),
		Chapter:                 chapter.NewRepository(db),
		Document:                document.NewRepository(db),
		DocumentFolder:          document_folder.NewRepository(db),
	}
}
