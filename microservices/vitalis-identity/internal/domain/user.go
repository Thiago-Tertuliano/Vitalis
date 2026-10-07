package domain

import "time"

const (
	StatusActive    = "active"
	StatusPending   = "pending_verification"
	StatusSuspended = "suspended"
	StatusDelete    = "deleted"
)

type User struct {
	ID             string
	Email          string
	Nome           string
	Telefone       string
	Status         string
	CPF            string
	DataNascimento *time.Time
	Roles          []string
	CreatedAt      time.Time
	UpdateAt       time.Time
}

func (u User) PodeReceberToken() bool { return u.Status == StatusActive }

func RoleValida(role string) bool {
	switch role {
	case "paciente", "medico", "farmacia", "motoboy", "admin":
		return true
	default:
		return false
	}
}
