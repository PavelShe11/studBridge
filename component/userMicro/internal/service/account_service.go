package service

import (
	"strings"
	"userMicro/internal/domain"
	"userMicro/internal/repository"

	"github.com/go-playground/validator/v10"
)

type AccountService struct {
	accountRepository *repository.AccountRepository
	validate          validator.Validate
}

func NewAccountService(accountRepository *repository.AccountRepository) *AccountService {
	return &AccountService{
		accountRepository: accountRepository,
	}
}

func (s *AccountService) CreateAccount(account domain.Account) *domain.Error {
	account.Email = strings.TrimSpace(account.Email)
	account.FirstName = strings.TrimSpace(account.FirstName)
	account.LastName = strings.TrimSpace(account.LastName)

	errs := s.ValidateAccountData(account)
	if errs != nil {
		return errs
	}
	err := s.accountRepository.CreateAccount(account)
	if err != nil {
		return &domain.Error{Error: "Internal Server Error"}
	}
	return nil
}

func (s *AccountService) GetAccountByEmail(email string) (*domain.Account, *domain.Error) {
	account, err := s.accountRepository.GetAccountByEmail(email)
	if err != nil {
		return nil, &domain.Error{Error: "Internal Server Error"}
	}
	return account, nil
}

func (s *AccountService) GetAccountById(id string) (*domain.Account, *domain.Error) {
	account, err := s.accountRepository.GetAccountById(id)
	if err != nil {
		return nil, &domain.Error{Error: "Internal Server Error"}
	}
	return account, nil
}

func (s *AccountService) ValidateAccountData(account domain.Account) *domain.Error {
	errs := &domain.Error{}
	if err := s.validate.Var(account.Email, "required,email"); err != nil {
		errs.FieldErrors = append(domain.ValidationErrorsMap(err.(validator.ValidationErrors)))
	}
	if err := s.validate.Var(account.FirstName, "required,firstName"); err != nil {
		errs.FieldErrors = append(domain.ValidationErrorsMap(err.(validator.ValidationErrors)))
	}
	if err := s.validate.Var(account.LastName, "required,lastName"); err != nil {
		errs.FieldErrors = append(domain.ValidationErrorsMap(err.(validator.ValidationErrors)))
	}
	if len(errs.FieldErrors) > 0 {
		errs.Error = "Ошибка валидации"
		return errs
	}
	return nil
}
