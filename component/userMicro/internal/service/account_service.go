package service

import (
	commondomain "github.com/PavelShe11/studbridge/common/domain"
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/user/internal/domain"
	"github.com/PavelShe11/studbridge/user/internal/repository"
	"github.com/PavelShe11/studbridge/user/utlis/validation"
)

type AccountService struct {
	accountRepository *repository.AccountRepository
	logger            logger.Logger
	validator         *validation.Validator
}

func NewAccountService(
	accountRepository *repository.AccountRepository,
	l logger.Logger,
	validator *validation.Validator,
) *AccountService {
	return &AccountService{
		accountRepository: accountRepository,
		logger:            l,
		validator:         validator,
	}
}

func (s *AccountService) CreateAccount(account domain.Account) error {
	errs := s.ValidateAccountData(account)
	if errs != nil {
		return errs
	}
	err := s.accountRepository.CreateAccount(account)
	if err != nil {
		s.logger.Error(err)
		return commondomain.InternalError
	}
	return nil
}

func (s *AccountService) GetAccountByEmail(email string) (*domain.Account, error) {
	account, err := s.accountRepository.GetAccountByEmail(email)
	if account == nil && err == nil {
		return nil, nil
	}
	if err != nil {
		s.logger.Error(err)
		return nil, commondomain.InternalError
	}
	return account, nil
}

func (s *AccountService) GetAccountById(id string) (*domain.Account, error) {
	account, err := s.accountRepository.GetAccountById(id)
	if err != nil {
		s.logger.Error(err)
		return nil, commondomain.InternalError
	}
	return account, nil
}

func (s *AccountService) ValidateAccountData(account domain.Account) error {
	errs := domain.ValidationError
	errs.FieldErrors = s.validator.Struct(&account)
	if len(errs.FieldErrors) > 0 {
		return errs
	}
	return nil
}
