package service

import (
	"strings"
	"userMicro/internal/domain"
	"userMicro/internal/repository"
	"userMicro/utlis/logger"
	"userMicro/utlis/validation"
)

type AccountService struct {
	accountRepository *repository.AccountRepository
	logger            logger.Logger
}

func NewAccountService(accountRepository *repository.AccountRepository, l logger.Logger) *AccountService {
	return &AccountService{
		accountRepository: accountRepository,
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
		return &domain.Error{Name: "internalServerError"}
	}
	return nil
}

func (s *AccountService) GetAccountByEmail(email string) (*domain.Account, *domain.Error) {
	account, err := s.accountRepository.GetAccountByEmail(email)
	if account == nil && err == nil {
		return nil, nil
	}
	if err != nil {
		s.logger.Error(err)
		return nil, &domain.Error{Name: "internalServerError"}
	}
	return account, nil
}

func (s *AccountService) GetAccountById(id string) (*domain.Account, *domain.Error) {
	account, err := s.accountRepository.GetAccountById(id)
	if err != nil {
		s.logger.Error(err)
		return nil, &domain.Error{Name: "internalServerError"}
	}
	return account, nil
}

func (s *AccountService) ValidateAccountData(account domain.Account) *domain.Error {
	errs := &domain.Error{
		Name: "validationError",
	}
	errs.FieldErrors = validation.Struct(&account)
	if len(errs.FieldErrors) > 0 {
		return errs
	}
	return nil
}
