package main

type User struct {
	ID          int64  `json:"id"`
	Issuer      string `json:"-"`
	Subject     string `json:"-"`
	Name        string `json:"name"`
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	Picture     string `json:"picture,omitempty"`
	IsDeveloper bool   `json:"isDeveloper"`
}

type PublicUser struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Username    string `json:"username,omitempty"`
	Picture     string `json:"picture,omitempty"`
	IsDeveloper bool   `json:"isDeveloper"`
}

func (u User) Public() PublicUser {
	return PublicUser{ID: u.ID, Name: u.Name, Username: u.Username, Picture: u.Picture, IsDeveloper: u.IsDeveloper}
}

type Votes struct {
	Want     int `json:"want"`
	DontWant int `json:"dontWant"`
}

type ViewerState struct {
	Liked bool   `json:"liked"`
	Vote  string `json:"vote,omitempty"`
}

type Blueprint struct {
	ID           int64       `json:"id"`
	Kind         string      `json:"kind"`
	Status       string      `json:"status"`
	Title        string      `json:"title"`
	Body         string      `json:"body"`
	Progress     int         `json:"progress"`
	Images       []string    `json:"images"`
	Author       PublicUser  `json:"author"`
	LikeCount    int         `json:"likeCount"`
	CommentCount int         `json:"commentCount"`
	Votes        Votes       `json:"votes"`
	Viewer       ViewerState `json:"viewer"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
}

type Update struct {
	ID           int64       `json:"id"`
	Title        string      `json:"title"`
	Body         string      `json:"body"`
	Author       PublicUser  `json:"author"`
	LikeCount    int         `json:"likeCount"`
	CommentCount int         `json:"commentCount"`
	Viewer       ViewerState `json:"viewer"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
}

type Comment struct {
	ID        int64      `json:"id"`
	Body      string     `json:"body"`
	Author    PublicUser `json:"author"`
	CreatedAt string     `json:"createdAt"`
}

type DailyLimits struct {
	CommentsRemaining int `json:"commentsRemaining"`
	RequestsRemaining int `json:"requestsRemaining"`
}

type CommunityBan struct {
	User      PublicUser `json:"user"`
	BannedBy  PublicUser `json:"bannedBy"`
	CreatedAt string     `json:"createdAt"`
}
