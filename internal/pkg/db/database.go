package db

import (
	"fmt"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/config"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/models/tasks"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/models/users"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"time"
)

var (
	DB  *gorm.DB
	err error
)

type Database struct {
	*gorm.DB
}

// SetupDB opens a database and saves the reference to `Database` struct.
func SetupDB() {
	var db = DB

	configuration := config.GetConfig()

	driver := configuration.Database.Driver
	database := configuration.Database.Dbname
	username := configuration.Database.Username
	password := configuration.Database.Password
	host := configuration.Database.Host
	port := configuration.Database.Port

	if driver == "sqlite" { // SQLITE
		db, err = gorm.Open("sqlite3", "./"+database+".db")
		if err != nil {
			fmt.Println("db err: ", err)
		}
	} else if driver == "postgres" { // POSTGRES
		db, err = gorm.Open("postgres", "host="+host+" port="+port+" user="+username+" dbname="+database+"  sslmode=disable password="+password)
		if err != nil {
			fmt.Println("db err: ", err)
		}
	} else if driver == "mysql" { // MYSQL
		db, err = gorm.Open("mysql", username+":"+password+"@tcp("+host+":"+port+")/"+database+"?charset=utf8&parseTime=True&loc=Local")
		if err != nil {
			fmt.Println("db err: ", err)
		}
	}

	// Change this to true if you want to see SQL queries
	db.LogMode(false)
	db.DB().SetMaxIdleConns(configuration.Database.MaxIdleConns)
	db.DB().SetMaxOpenConns(configuration.Database.MaxOpenConns)
	db.DB().SetConnMaxLifetime(time.Duration(configuration.Database.MaxLifetime) * time.Second)
	DB = db
	migration()
}

// Auto migrate project models
func migration() {
	DB.AutoMigrate(&users.User{})
	DB.AutoMigrate(&users.UserRole{})
	DB.AutoMigrate(&tasks.Task{})
}

func GetDB() *gorm.DB {
	return DB
}
