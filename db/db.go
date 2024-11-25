package db

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"moniter/conf"
)

var DBConn *gorm.DB

func Init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		conf.Sc.DB.User, conf.Sc.DB.Password, conf.Sc.DB.Host, conf.Sc.DB.Port, conf.Sc.DB.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	DBConn = db
}
