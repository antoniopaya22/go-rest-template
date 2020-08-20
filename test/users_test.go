package test

import (
	"fmt"
	"github.com/antonioalfa22/GoGin-API-REST-Template/configs"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/daos"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/services"
	"testing"
)

var userTest models.User

func Setup() {
	configs.Setup()
	configs.SetupDB()
	configs.GetDB().Exec("DELETE FROM users")
}

func TestAddUser(t *testing.T)  {
	Setup()
	user := models.User{
		Firstname: "Antonio",
		Lastname: "Paya",
		Username: "antonio",
		Hash: "hash",
	}
	s := services.NewUserService(daos.NewUserDAO())
	if err := s.Add(&user); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	userTest = user
}

func TestGetAllUsers(t *testing.T) {
	s := services.NewUserService(daos.NewUserDAO())
	if _, err := s.All(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestGetUserById(t *testing.T)  {
	configs.SetupDB()
	configs.SetupDB()
	s := services.NewUserService(daos.NewUserDAO())
	if _, err := s.Get(fmt.Sprint(userTest.ID)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}