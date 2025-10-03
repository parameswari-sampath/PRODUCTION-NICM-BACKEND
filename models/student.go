package models

import "time"

type Student struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateStudentRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateStudentRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
