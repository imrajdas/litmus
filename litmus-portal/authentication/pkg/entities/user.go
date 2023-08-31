package entities

import (
	"net/mail"
)

// Role states the role of the user in the portal
type Role string

const (
	// RoleAdmin gives the admin permissions to a user
	RoleAdmin Role = "admin"

	//RoleUser gives the normal user permissions to a user
	RoleUser Role = "user"
)

// User contains the user information
type User struct {
	ID            string  `bson:"_id,omitempty" json:"_id"`
	UserName      string  `bson:"username,omitempty" json:"username"`
	Password      string  `bson:"password,omitempty" json:"password,omitempty"`
	Email         string  `bson:"email,omitempty" json:"email,omitempty"`
	Name          string  `bson:"name,omitempty" json:"name,omitempty"`
	Role          Role    `bson:"role,omitempty" json:"role"`
	CreatedAt     *string `bson:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt     *string `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
	DeactivatedAt *string `bson:"deactivated_at,omitempty" json:"deactivated_at,omitempty"`
}

// UserDetails is used to update user's personal details
type UserDetails struct {
	ID       string `bson:"id,omitempty"`
	Email    string `bson:"email,omitempty" json:"email,omitempty"`
	Name     string `bson:"name,omitempty" json:"name,omitempty"`
	Password string `bson:"password,omitempty" json:"password,omitempty"`
}

// UserPassword defines structure for password related requests
type UserPassword struct {
	Username    string `json:"username,omitempty"`
	OldPassword string `json:"old_password,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}

// UpdateUserState defines structure to deactivate or reactivate user
type UpdateUserState struct {
	Username     string `json:"username"`
	IsDeactivate *bool  `json:"is_deactivate"`
}

// APIStatus defines structure for APIroute status
type APIStatus struct {
	Status string `json:"status"`
}

type UserWithProject struct {
	ID        string     `bson:"_id"`
	Username  string     `bson:"username"`
	CreatedAt string     `bson:"created_at"`
	Email     string     `bson:"email"`
	Name      string     `bson:"name"`
	Projects  []*Project `bson:"projects"`
}

func (user User) GetUserWithProject() *UserWithProject {

	return &UserWithProject{
		ID:        user.ID,
		Username:  user.UserName,
		Name:      user.Name,
		CreatedAt: *user.CreatedAt,
		Email:     user.Email,
	}
}

// SanitizedUser returns the user object without sensitive information
func (user *User) SanitizedUser() *User {
	user.Password = ""
	return user
}

// IsEmailValid validates the email
func (user *User) IsEmailValid(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
