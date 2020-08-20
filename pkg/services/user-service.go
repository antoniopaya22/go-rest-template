package services

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/models"
	"github.com/gin-gonic/gin"
)

type userDAO interface {
	Get(id string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	All() (*[]models.User, error)
	Add(user *models.User) error
	Update(user *models.User) error
	Delete(user *models.User) error
	AllPaginated(c *gin.Context) (*models.UserData, error)
}

type UserService struct {
	dao userDAO
}

// NewUserService creates a new UserService with the given user DAO.
func NewUserService(dao userDAO) *UserService {
	return &UserService{dao}
}

func (s *UserService) Get(id string) (*models.User, error) { return s.dao.Get(id) }
func (s *UserService) GetByUsername(username string) (*models.User, error) { return s.dao.GetByUsername(username) }
func (s *UserService) All() (*[]models.User, error) { return s.dao.All() }
func (s *UserService) Add(user *models.User) error { return s.dao.Add(user) }
func (s *UserService) Update(user *models.User) error { return s.dao.Update(user) }
func (s *UserService) Delete(user *models.User) error { return s.dao.Delete(user) }
func (s *UserService) AllPaginated(c *gin.Context) (*models.UserData, error) { return s.dao.AllPaginated(c) }