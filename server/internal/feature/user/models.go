package user

import (
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/subject"
)

type UserScopeResponse struct {
	Scopes []subject.Scope `json:"scopes"`
}

type UserResponse struct {
	User sqlc.User `json:"user"`
}

func ConvertUserResponse(user *sqlc.User) UserResponse {
	if user == nil {
		return UserResponse{}
	}
	return UserResponse{User: *user}
}

type Message struct {
	UserId  uuid.UUID `json:"userId"`
	Message string    `json:"message"`
}
