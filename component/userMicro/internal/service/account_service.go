package service

import (
	"strings"
	"userMicro/internal/domain"
	"userMicro/internal/repository"
	"userMicro/utlis/logger"

	"github.com/go-playground/validator/v10"
)

type AccountService struct {
	accountRepository *repository.AccountRepository
	validate          validator.Validate
	logger            logger.Logger
}

func NewAccountService(accountRepository *repository.AccountRepository, l logger.Logger) *AccountService {
	return &AccountService{
		accountRepository: accountRepository,
		validate:          *validator.New(),
		logger:            l,
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
		s.logger.Error(err)
		return &domain.Error{Name: "internalServerError1"}
	}
	return nil
}

func (s *AccountService) GetAccountByEmail(email string) (*domain.Account, *domain.Error) {
	account, err := s.accountRepository.GetAccountByEmail(email)
	if err != nil {
		s.logger.Error(err)
		return nil, &domain.Error{Name: "internalServerError2"}
	}
	return account, nil
}

func (s *AccountService) GetAccountById(id string) (*domain.Account, *domain.Error) {
	account, err := s.accountRepository.GetAccountById(id)
	if err != nil {
		s.logger.Error(err)
		return nil, &domain.Error{Name: "internalServerError3"}
	}
	return account, nil
}

func (s *AccountService) ValidateAccountData(account domain.Account) *domain.Error {
	errs := &domain.Error{
		Name: "validationError",
	}
	if err := s.validate.Var(account.Email, "required,email"); err != nil {
		errs.FieldErrors = append(errs.FieldErrors, domain.ValidationErrorsMap(err.(validator.ValidationErrors))...)
	}
	if err := s.validate.Var(account.FirstName, "required,min=2,max=50"); err != nil {
		errs.FieldErrors = append(errs.FieldErrors, domain.ValidationErrorsMap(err.(validator.ValidationErrors))...)
	}
	if err := s.validate.Var(account.LastName, "required,min=2,max=50"); err != nil {
		errs.FieldErrors = append(errs.FieldErrors, domain.ValidationErrorsMap(err.(validator.ValidationErrors))...)
	}
	if len(errs.FieldErrors) > 0 {
		return errs
	}
	return nil
}
