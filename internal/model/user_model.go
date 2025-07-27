package model

import (
	"time"

	"gorm.io/gorm"
)

// UserRole represents different user privilege levels
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleCrawler UserRole = "crawler"
	RoleWorker  UserRole = "worker"
	RoleUser    UserRole = "user"
)

// User represents a registered user in the system.
type User struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	Email     string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"`
	Role      UserRole       `gorm:"type:varchar(50);not null;default:'user'" json:"role"`
	URLs      []URL          `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"urls,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// UserDTO is used for sending user data in HTTP responses.
type UserDTO struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the name of the table for User.
func (User) TableName() string {
	return "users"
}

// CreateUserInput defines expected fields for creating a user.
type CreateUserInput struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=6"`
	Role     UserRole `json:"role,omitempty"` // Optional, defaults to "user"
}

// ToDTO converts the User model into a UserDTO for responses.
func (u *User) ToDTO() *UserDTO {
	return &UserDTO{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// FromCreateInput maps CreateUserInput to the User model.
func UserFromCreateInput(input *CreateUserInput) *User {
	timeNow := time.Now()
	role := input.Role
	if role == "" {
		role = RoleUser // Default role
	}

	return &User{
		Username:  input.Username,
		Email:     input.Email,
		Password:  input.Password,
		Role:      role,
		CreatedAt: timeNow,
		UpdatedAt: timeNow,
	}
}

// Role-based permission methods
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsCrawler() bool {
	return u.Role == RoleCrawler || u.IsAdmin()
}

func (u *User) IsWorker() bool {
	return u.Role == RoleWorker || u.IsCrawler()
}

func (u *User) CanManageUsers() bool {
	return u.IsAdmin()
}

func (u *User) CanStartCrawls() bool {
	return u.IsCrawler()
}

func (u *User) CanProcessJobs() bool {
	return u.IsWorker()
}
