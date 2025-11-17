package intune

// User represents an Entra ID user.
type User struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	UserPrincipalName string `json:"userPrincipalName"`
	MailNickname      string `json:"mailNickname"`
	Department        string `json:"department"`
}

// Group represents an Entra ID group.
type Group struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// UserProfile contains user information and group memberships.
type UserProfile struct {
	User   *User
	Groups []Group
}
