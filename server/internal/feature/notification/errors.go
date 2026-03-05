package notification

import (
	"errors"
	"net/http"

	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/respond"
)

var (
	ErrNotificationNotFound     = errors.New("notification not found")
	ErrNotificationAccessDenied = errors.New("access denied to this notification")
	ErrInvalidNotificationID    = errors.New("invalid notification id")
)

func mapError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotificationNotFound) {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	if errors.Is(err, ErrNotificationAccessDenied) {
		respond.Error(w, apierror.Forbidden())
		return
	}
	respond.Error(w, apierror.Internal())
}
