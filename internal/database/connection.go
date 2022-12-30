package database

import (
	"log"
	"strings"

	"github.com/zerobit-tech/godbc/database/sql"

	_ "github.com/zerobit-tech/godbc"
)

type ColumnType struct {
	Name string

	HasNullable       bool
	HasLength         bool
	HasPrecisionScale bool

	Nullable     bool
	Length       int64
	DatabaseType string
	Precision    int64
	Scale        int64
}

type DBServer interface {
	GetConnectionID() string
	GetConnectionType() string
	GetConnectionString() string
}

var connectionMap map[string]*sql.DB = make(map[string]*sql.DB)

func ClearCache(server DBServer) {
	delete(connectionMap, server.GetConnectionID())
}

func GetConnection(server DBServer) (*sql.DB, error) {
	connectionID := server.GetConnectionID()
	db, found := connectionMap[connectionID]
	if found && db != nil {
		return db, nil
	}

	//fmt.Println((" ========================== BUILDING NEW CONNECTION ===================================="))
	db, err := sql.Open(strings.ToLower(server.GetConnectionType()), server.GetConnectionString())

	if err == nil {
		connectionMap[connectionID] = db

	} else {
		log.Println(" connetion errror 1>>>>>>>>>>>>", err)
	}

	//db.Ping()

	return db, err
}

func GetSingleConnection(server DBServer) (*sql.DB, error) {

	db, err := sql.Open(strings.ToLower(server.GetConnectionType()), server.GetConnectionString())

	db.SetMaxOpenConns(1)

	return db, err
}
