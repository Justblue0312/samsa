package subject

type Scope string

const (
	WebReadScope  Scope = "web:read"
	WebWriteScope Scope = "web:write"

	ProfileReadScope  Scope = "profile:read"
	ProfileWriteScope Scope = "profile:write"

	AuthorReadScope  Scope = "author:read"
	AuthorWriteScope Scope = "author:write"

	FileReadScope  Scope = "file:read"
	FileWriteScope Scope = "file:write"

	SubmissionReadScope  Scope = "submission:read"
	SubmissionWriteScope Scope = "submission:write"

	// Story scopes
	StoryReadScope  Scope = "story:read"
	StoryWriteScope Scope = "story:write"

	// Story vote scopes
	StoryVoteReadScope  Scope = "story.vote:read"
	StoryVoteWriteScope Scope = "story.vote:write"

	// Story report scopes
	StoryReportReadScope  Scope = "story.report:read"
	StoryReportWriteScope Scope = "story.report:write"

	// Story post scopes
	StoryPostReadScope  Scope = "story.post:read"
	StoryPostWriteScope Scope = "story.post:write"

	// Tag scopes
	TagReadScope  Scope = "tag:read"
	TagWriteScope Scope = "tag:write"

	// Comment scopes
	CommentReadScope     Scope = "comment:read"
	CommentWriteScope    Scope = "comment:write"
	CommentModerateScope Scope = "comment:moderate"

	// Chapter scopes
	ChapterReadScope  Scope = "chapter:read"
	ChapterWriteScope Scope = "chapter:write"

	// Document scopes
	DocumentReadScope  Scope = "document:read"
	DocumentWriteScope Scope = "document:write"

	// Notification scopes
	NotificationReadScope  Scope = "notification:read"
	NotificationWriteScope Scope = "notification:write"
)

var DefaultScopes = []Scope{WebReadScope, WebWriteScope}

func StrToScope(s []string) []Scope {
	var scopes []Scope
	for _, sc := range s {
		scopes = append(scopes, Scope(sc))
	}
	return scopes
}

func ScopeToStr(scopes []Scope) []string {
	var str []string
	for _, scope := range scopes {
		str = append(str, string(scope))
	}
	return str
}
