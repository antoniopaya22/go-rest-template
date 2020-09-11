package services

import (
	models "github.com/antonioalfa22/go-rest-template/internal/pkg/models/tasks"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/persistence"
)

type taskDAO interface {
	Get(id string) (*models.Task, error)
	All() (*[]models.Task, error)
	Add(t *models.Task) error
	Update(t *models.Task) error
	Delete(t *models.Task) error
	Query(q *models.Task) (*[]models.Task, error)
}

type TaskService struct {
	dao taskDAO
}

var taskService *TaskService

func GetTaskService() *TaskService {
	if taskService == nil {
		taskService = &TaskService{persistence.NewTaskDAO()}
	}
	return taskService
}

func (s *TaskService) Get(id string) (*models.Task, error)          { return s.dao.Get(id) }
func (s *TaskService) All() (*[]models.Task, error)                 { return s.dao.All() }
func (s *TaskService) Add(t *models.Task) error                     { return s.dao.Add(t) }
func (s *TaskService) Update(t *models.Task) error                  { return s.dao.Update(t) }
func (s *TaskService) Delete(t *models.Task) error                  { return s.dao.Delete(t) }
func (s *TaskService) Query(q *models.Task) (*[]models.Task, error) { return s.dao.Query(q) }
