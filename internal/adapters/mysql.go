package adapters

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/lmtani/pumbaa/internal/types"
)

type Mysql struct {
	Host     string
	Port     int
	User     string
	Password string
}

func NewMysql(db types.Database) *Mysql {
	return &Mysql{
		Host:     db.Host,
		Port:     db.Port,
		User:     db.User,
		Password: db.Password,
	}
}

func (m *Mysql) CheckConnection() error {
	dbConn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", m.User, m.Password, m.Host, m.Port)
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
