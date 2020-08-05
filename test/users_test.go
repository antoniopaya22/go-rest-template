package test

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/models"
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/repository"
	"testing"
)

func TestGetAllUsers(t *testing.T) {
	var users []models.User
	if err := repository.AllUsers(&users); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestGetUserById(t *testing.T)  {
	id := "1"
	var user models.User
	if err := repository.FindUserById(&user, id); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}