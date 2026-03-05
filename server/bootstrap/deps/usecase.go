package deps

import (
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/feature/chapter"
	"github.com/justblue/samsa/internal/feature/document"
	document_folder "github.com/justblue/samsa/internal/feature/document_folder"
	"github.com/justblue/samsa/internal/feature/flag"
	"github.com/justblue/samsa/internal/feature/genre"
	"github.com/justblue/samsa/internal/feature/notification"
	"github.com/justblue/samsa/internal/feature/spinnet"
	"github.com/justblue/samsa/internal/feature/story"
	"github.com/justblue/samsa/internal/feature/user"
	"github.com/justblue/samsa/internal/infras/cache"
	"github.com/justblue/samsa/internal/infras/redis"
	"github.com/justblue/samsa/internal/transport/ws"
)

type UseCases struct {
	User           user.UseCase
	Notification   notification.UseCase
	Flag           flag.UseCase
	Spinnet        spinnet.UseCase
	Story          story.UseCase
	Genre          genre.UseCase
	Chapter        chapter.UseCase
	Document       document.UseCase
	DocumentFolder document_folder.UseCase
}

func NewUseCases(cfg *config.Config, cacheClient *cache.Client, repos *Repositories, presenceStore *redis.PresenceStore, wsPublisher *ws.Publisher) *UseCases {
	notifier := notification.NewNotifier(wsPublisher)
	return &UseCases{
		User:           user.NewUseCase(cfg, cacheClient, repos.User, repos.OAuthAccount, presenceStore),
		Notification:   notification.NewUseCase(cfg, wsPublisher, notifier, repos.Notification, repos.NotificationRecipient),
		Flag:           flag.NewUseCase(repos.Flag),
		Spinnet:        spinnet.NewUseCase(repos.Spinnet),
		Story:          story.NewUseCase(repos.Story),
		Genre:          genre.NewUseCase(repos.Genre),
		Chapter:        chapter.NewUseCase(repos.Chapter),
		Document:       document.NewUseCase(repos.Document),
		DocumentFolder: document_folder.NewUseCase(repos.DocumentFolder),
	}
}
