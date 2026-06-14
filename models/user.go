package models

import "time"

// User représente un utilisateur (administrateur ou modérateur) de l'application
type User struct {
	ID         int       `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Role       string    `json:"role"` // 'admin' ou 'moderator'
	PictureURL string    `json:"picture_url"`
	CreatedAt  time.Time `json:"created_at"`
}
