package deps

import (
	"github.com/go-playground/validator/v10"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/feature/chapter"
	"github.com/justblue/samsa/internal/feature/document"
	document_folder "github.com/justblue/samsa/internal/feature/document_folder"
	"github.com/justblue/samsa/internal/feature/flag"
	"github.com/justblue/samsa/internal/feature/genre"
	"github.com/justblue/samsa/internal/feature/integrations/ws/cursor"
	"github.com/justblue/samsa/internal/feature/integrations/ws/notification"
	"github.com/justblue/samsa/internal/feature/integrations/ws/presence"
	"github.com/justblue/samsa/internal/feature/integrations/ws/typing"
	"github.com/justblue/samsa/internal/feature/spinnet"
	"github.com/justblue/samsa/internal/feature/story"
	"github.com/justblue/samsa/internal/feature/user"
	httpTransport "github.com/justblue/samsa/internal/transport/http"
	wsTransport "github.com/justblue/samsa/internal/transport/ws"
)

// NewHTTPHandlers constructs all HTTP feature handlers.
// Add new feature handlers here when extending the API.
func NewHTTPHandlers(uCase *UseCases, cfg *config.Config, v *validator.Validate) *httpTransport.HTTPHandlers {
	return &httpTransport.HTTPHandlers{
		User:           user.NewHTTPHandler(uCase.User, cfg, v),
		Flag:           flag.NewHTTPHandler(uCase.Flag, cfg, v),
		Spinnet:        spinnet.NewHTTPHandler(uCase.Spinnet, cfg, v),
		Story:          story.NewHTTPHandler(uCase.Story, cfg, v),
		Genre:          genre.NewHTTPHandler(uCase.Genre, v),
		Chapter:        chapter.NewHTTPHandler(uCase.Chapter, cfg, v),
		Document:       document.NewHTTPHandler(uCase.Document, cfg, v),
		DocumentFolder: document_folder.NewHTTPHandler(uCase.DocumentFolder, cfg, v),
	}
}

// NewWSHandlers constructs all WebSocket feature handlers and returns them
// as a *wsTransport.WSHandlers ready to be registered into the hub's Registry.
// Add new feature handlers here when extending the WebSocket API.
func NewWSHandlers(
	presenceStore presence.PresenceStore,
	typingStore typing.TypingStore,
	hub *wsTransport.Hub,
	usecases *UseCases,
) *wsTransport.WSHandlers {
	h := &wsTransport.WSHandlers{}
	h.Add(
		presence.NewPresenceHandler(presenceStore, hub),
		notification.NewNotificationHandler(usecases.Notification),
		typing.NewTypingHandler(typingStore, hub),
		cursor.NewCursorHandler(hub),
	)
	return h
}
