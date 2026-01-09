package entity

type Account struct {
	Id        string
	FirstName string `validate:"required,min=2,max=50"`
	LastName  string `validate:"required,min=2,max=50"`
	Email     string `validate:"required,email,min=6,max=100"`
}
