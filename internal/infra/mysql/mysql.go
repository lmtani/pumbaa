package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Mysql struct {
	Host     string
	Port     string
	User     string
	Password string
}

func NewMysql(host, port, user, password string) *Mysql {
	return &Mysql{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
}

func (m *Mysql) CheckConnection() error {
	dbConn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", m.User, m.Password, m.Host, m.Port)
	db, err := sql.Open("mysql", dbConn)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println("Failed to close database connection:", err)
		}
	}(db)

	err = db.Ping()
	if err != nil {
		return err
	}
	return nil
}
