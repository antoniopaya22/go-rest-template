package persistence

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/db"
	models "github.com/antonioalfa22/GoGin-API-REST-Template/internal/pkg/models/tasks"
	"strconv"
)

// TaskDAO persists task data in database
type TaskDAO struct{}

// NewTaskDAO creates a new TaskDAO
func NewTaskDAO() *TaskDAO {
	return &TaskDAO{}
}

func (dao *TaskDAO) Get(id string) (*models.Task, error) {
	var task models.Task
	where := models.Task{}
	where.ID, _ = strconv.ParseUint(id, 10, 64)
	_, err := First(&where, &task, []string{"User"})
	if err != nil {
		return nil, err
	}
	return &task, err
}

func (dao *TaskDAO) All() (*[]models.Task, error) {
	var tasks []models.Task
	err := Find(&models.Task{}, &tasks, []string{"User"}, "id asc")
	return &tasks, err
}

func (dao *TaskDAO) Query(q *models.Task) (*[]models.Task, error) {
	var tasks []models.Task
	err := Find(&q, &tasks, []string{"User"}, "id asc")
	return &tasks, err
}

func (dao *TaskDAO) Add(task *models.Task) error {
	err := Create(&task)
	err = Save(&task)
	return err
}

func (dao *TaskDAO) Update(task *models.Task) error { return db.GetDB().Omit("User").Save(&task).Error }

func (dao *TaskDAO) Delete(task *models.Task) error { return db.GetDB().Unscoped().Delete(&task).Error }
