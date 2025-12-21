package domain

type Account struct {
	Id        string `json:"id" db:"id"`
	FirstName string `json:"firstName" db:"first_name" validate:"required,min=2,max=50"`
	LastName  string `json:"lastName" db:"last_name" validate:"required,min=2,max=50"`
	Email     string `json:"email" db:"email" validate:"required,email,min=6,max=100"`
}
