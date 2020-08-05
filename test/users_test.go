package test

import (
	"fmt"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/repository"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/config"
	"github.com/antonioalfa22/GoGin-API-REST-Template/pkg/database"
	"testing"
)

var userTest models.User

func Setup() {
	config.Setup()
	database.Setup()
	database.GetDB().Exec("DELETE FROM users")
}

func TestAddUser(t *testing.T)  {
	Setup()
	user := models.User{
		Firstname: "Antonio",
		Lastname: "Paya",
		Username: "antonio",
		Hash: "hash",
	}
	if err := repository.AddUser(&user); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	userTest = user
}

func TestGetAllUsers(t *testing.T) {
	var users []models.User
	if err := repository.AllUsers(&users); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("Users len expected == 1")
	}
}

func TestGetUserById(t *testing.T)  {
	config.Setup()
	database.Setup()
	var user models.User
	if err := repository.FindUserById(&user, fmt.Sprint(userTest.ID)); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}