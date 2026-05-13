package models

type LoginRequest struct {
	PhoneNumber string `json:"phone_number" db:"phone_number" binding:"required"`
	Password    string `json:"password" db:"password" binding:"required"`
}

type User struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	PhoneNumber string `json:"phone_number" db:"phone_number"`
	Password    string `json:"password" db:"password"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

type UserRequest struct {
	Name string `json:"name" db:"name" binding:"required,min=3"`
	Password string `json:"password" db:"password" binding:"required,min=6"`
	PhoneNumber string `json:"phone_number" db:"phone_number" binding:"required"`
}