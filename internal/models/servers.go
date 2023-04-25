package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/database"
	"github.com/onlysumitg/qsql2/internal/validator"
	"github.com/zerobit-tech/godbc"

	bolt "go.etcd.io/bbolt"
)

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new User type. Notice how the field names and types align
// with the columns in the database "users" table?
type Server struct {
	ID   string `json:"id" db:"id" form:"id"`
	Name string `json:"server_name" db:"server_name" form:"name"`
	IP   string `json:"ip" db:"ip" form:"ip"`
	Port uint8  `json:"port" db:"port" form:"port"`
	Ssl  bool   `json:"ssl" db:"ssl" form:"ssl"`

	UserName    string    `json:"user_name" db:"user_name" form:"user_name"`
	Password    string    `json:"password" db:"password" form:"password"`
	WorkLib     string    `json:"worklib" db:"worklib" form:"worklib"`
	CreatedAt   time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at" form:"-"`
	Connections uint      `json:"connections" db:"connections" form:"connections"`
	AllowInsert bool      `json:"allowinsert" db:"allowinsert" form:"allowinsert"`
	AllowUpdate bool      `json:"allowupdate" db:"allowupdate" form:"allowupdate"`
	AllowDelete bool      `json:"allowdelete" db:"allowdelete" form:"allowdelete"`

	OnHold        bool   `json:"onhold" db:"onhold" form:"onhold"`
	OnHoldMessage string `json:"onholdmessage" db:"onholdmessage" form:"onholdmessage"`

	validator.Validator
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) GetConnectionString() string {
	driver := "IBM i Access ODBC Driver"
	ssl := 0
	if s.Ssl {
		ssl = 1
	}
	//connectionString := fmt.Sprintf("DRIVER=%s;SYSTEM=%s; UID=%s;PWD=%s;DBQ=*USRLIBL;UNICODESQL=1;XDYNAMIC=1;EXTCOLINFO=1;PKG=A/DJANGO,2,0,0,1,512;PROTOCOL=TCPIP;NAM=1;CMT=0;SSL=%d;ALLOWUNSCHAR=1", driver, s.IP, s.UserName, s.Password, ssl)
	connectionString := fmt.Sprintf("DRIVER=%s;SYSTEM=%s; UID=%s;PWD=%s;DBQ=*USRLIBL;UNICODESQL=1;XDYNAMIC=1;EXTCOLINFO=1;PKG=A/DJANGO,2,0,0,1,512;PROTOCOL=TCPIP;NAM=1;CMT=0;SSL=%d;ALLOWUNSCHAR=1", driver, s.IP, s.UserName, s.Password, ssl)

	return connectionString
}

func (s Server) GetConnectionType() string {
	return "odbc"
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) GetConnectionID() string {
	return s.ID
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) ClearCache() {
	database.ClearCache(s)
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) GetConnection() (*sql.DB, error) {
	if s.OnHold {
		return nil, fmt.Errorf("Server is on hold due to %s", s.OnHoldMessage)
	}

	return database.GetConnection(s)
}
func (s Server) GetSinglaConnection() (*sql.DB, error) {
	if s.OnHold {
		return nil, fmt.Errorf("Server is on hold due to %s", s.OnHoldMessage)
	}

	return database.GetSingleConnection(s)
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s *Server) RunQuery(runningSQL *RunningSql) (queryResults []*QueryResult) {

	if runningSQL.Error != "" {
		return_rows := make([]map[string]interface{}, 0)
		column_types := make([]database.ColumnType, 0)
		queryResults = append(queryResults, &QueryResult{CurrentSql: *runningSQL, Rows: return_rows, Columns: column_types, Heading: "Error", ErrorMessage: runningSQL.Error})
		return queryResults
	}

	switch runningSQL.StatementType {
	case "INSERT":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "UPDATE":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "DELETE":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "CREATE":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "COMMIT":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "ROLLBACK":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "CALL":
		queryResults = s.CallSP(runningSQL)
	case "@BATCH":
		queryResults = s.BatchStatement(runningSQL)
	case "@DOWNLOAD":
		queryResults = s.DownloadData(runningSQL)
	default:
		queryResults = s.RunSelectQuery(runningSQL)
	}

	for _, queryResult := range queryResults {
		queryResult.CurrentSql = *runningSQL
	}
	return
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) DownloadData(runningSQL *RunningSql) (queryResults []*QueryResult) {
	return_rows := make([]map[string]interface{}, 0)

	column_types := make([]database.ColumnType, 0)

	column_type := database.ColumnType{
		IndexName: "Download_Link",
		Name:      "Download_Link",
		IsLink:    true,
	}
	column_types = append(column_types, column_type)
	row := make(map[string]interface{})

	queryResultsTemp := s.RunSelectQuery(runningSQL)

	fileName := queryResultsTemp[0].ToExcel()

	row["Download_Link"] = fmt.Sprintf("/downloadexcel/%s", fileName)
	return_rows = append(return_rows, row)
	queryResult := &QueryResult{CurrentSql: *runningSQL,
		Rows: return_rows, Columns: column_types,
		Heading:      "Result",
		FlashMessage: "",
	}

	queryResults = append(queryResults, queryResult)

	return
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) BatchStatement(runningSQL *RunningSql) (queryResults []*QueryResult) {
	return_rows := make([]map[string]interface{}, 0)
	column_types := make([]database.ColumnType, 0)
	runningSQL.LoadMore = false

	column_type := database.ColumnType{
		IndexName: "Job",
		Name:      "Job",
	}
	column_typeID := database.ColumnType{
		IndexName: "Job ID",
		Name:      "Job ID",
	}

	row := make(map[string]interface{})
	row["Job"] = "Batch job has been submitted."
	row["Job ID"] = runningSQL.ID

	return_rows = append(return_rows, row)
	column_types = append(column_types, column_type)

	column_types = append(column_types, column_typeID)

	queryResult := &QueryResult{CurrentSql: *runningSQL,
		Rows: return_rows, Columns: column_types,
		Heading:      "Result",
		FlashMessage: "",
	}

	queryResults = append(queryResults, queryResult)
	//batchSql.
	return

}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s *Server) isAllowed(runningSQL *RunningSql) bool {
	switch runningSQL.StatementType {
	case "INSERT":
		return s.AllowInsert
	case "UPDATE":
		return s.AllowUpdate
	case "DELETE":
		return s.AllowDelete

	default:
		return true
	}

}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s *Server) IsValid(runningSQL *RunningSql) error {

	if s.OnHold {
		return fmt.Errorf("Server is on hold due to %s", s.OnHoldMessage)
	}

	switch runningSQL.StatementType {

	case "UPDATE": // must have a where clause
		match, err := regexp.MatchString(`(?mi)\s+where\s+`, runningSQL.RunningNow)

		if err != nil {
			match = false
		}
		if !match {
			return errors.New("Where clause is required for Update statements")

		}

	case "DELETE": // must have a where clause
		_, err := regexp.MatchString(`(?mi)\s+where\s+`, runningSQL.RunningNow)
		if err != nil {
			return errors.New("Where clause is required for Delete statements")
		}

	}
	return nil
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s *Server) RunExecuteQuery(runningSQL *RunningSql) (queryResults []*QueryResult) {
	return_rows := make([]map[string]interface{}, 0)
	column_types := make([]database.ColumnType, 0)

	if !s.isAllowed(runningSQL) {
		errorMessge := fmt.Sprintf("%s not allowed. Please update server %s to allow %s.", runningSQL.StatementType, s.Name, runningSQL.StatementType)
		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Not Allowed", ErrorMessage: errorMessge})

		return
	}

	notvalidErr := s.IsValid(runningSQL)
	if notvalidErr != nil {
		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Onhold", ErrorMessage: notvalidErr.Error()})

		return
	}

	conn, errConnection := s.GetConnection()

	if errConnection != nil {
		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Connection Error", ErrorMessage: errConnection.Error()})

		return
	}

	res, err_query := conn.Exec(runningSQL.RunningNow) //"select * from sumitg1/qsqltest")
	if err_query != nil {
		var odbcError *godbc.Error

		if errors.As(err_query, &odbcError) {
			s.UpdateAfterError(odbcError)
		}

		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Query Error", ErrorMessage: err_query.Error()})
		return
	}

	count, err2 := res.RowsAffected()
	if err2 != nil {
		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Error", ErrorMessage: err2.Error()})
		return
	}
	runningSQL.LoadMore = false
	// return_rows, column_types = database.ToMap(rows_to_process, 10, runningSQL.ScrollTo)
	column_type := database.ColumnType{
		IndexName: "Rows Impacted",
		Name:      "Rows Impacted",
	}

	row := make(map[string]interface{})
	row["Rows Impacted"] = count

	return_rows = append(return_rows, row)
	column_types = append(column_types, column_type)

	queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: runningSQL.StatementType})

	return

}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s *Server) UpdateAfterError(odbcError *godbc.Error) (retry bool) {

	if strings.EqualFold(odbcError.APIName, "SQLDriverConnect") {
		for _, diag := range odbcError.Diag {
			switch diag.NativeError {
			case 8001:
				s.OnHold = true
				s.OnHoldMessage = "User name does not exist."

			case 8002:
				s.OnHold = true
				s.OnHoldMessage = "Incorrect password"

			case 30189: // connection login timeout HYT000

				retry = true
			}
		}
	}

	return
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s *Server) RunSelectQuery(runningSQL *RunningSql) (queryResults []*QueryResult) {
	return_rows := make([]map[string]interface{}, 0)
	column_types := make([]database.ColumnType, 0)

	notvalidErr := s.IsValid(runningSQL)
	if notvalidErr != nil {
		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Onhold", ErrorMessage: notvalidErr.Error()})

		return
	}

	openQuery, ok := runningSQLQueryMap[runningSQL.ID]
	rows_to_process := openQuery.Query

	if !ok {
		conn, err_connection := s.GetConnection()

		if err_connection != nil {
			queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Connection Error", ErrorMessage: err_connection.Error()})
			return
		}

		ctx := context.Background()
		ctx = context.WithValue(ctx, godbc.LABEL_IN_COL_NAME, true)
		rows, err_query := conn.QueryContext(ctx, runningSQL.RunningNow) //conn.Query(runningSQL.RunningNow) //"select * from sumitg1/qsqltest")
		if err_query != nil {
			var odbcError *godbc.Error

			/// need to make sure connection is good to use

			log.Printf(" connetion errror 2>>>>>>>>>>>>%t", err_query)
			if errors.As(err_query, &odbcError) {

				// retry := s.UpdateAfterError(odbcError)
			}

			queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Query Error", ErrorMessage: err_query.Error()})
			return
		}

		rows_to_process = rows

	}

	return_rows, column_types = database.ToMap(rows_to_process, runningSQL.ResultSetSize, runningSQL.ScrollTo)

	recordsProcessed := len(return_rows)
	if recordsProcessed >= runningSQL.ResultSetSize {
		runningSQL.LoadMore = true
		runningSQLQueryMap[runningSQL.ID] = OpenQuery{Query: rows_to_process, SessionID: runningSQL.SessionID}
	}

	if runningSQL.LoadMore && len(return_rows) < runningSQL.ResultSetSize {

		query, found := runningSQLQueryMap[runningSQL.ID]
		if found {
			query.Query.Close()
		}

		delete(runningSQLQueryMap, runningSQL.ID)
		defer rows_to_process.Close()
		runningSQL.LoadMore = false
	}

	if !runningSQL.LimitRecods {
		query, found := runningSQLQueryMap[runningSQL.ID]
		if found {
			query.Query.Close()
		}

		delete(runningSQLQueryMap, runningSQL.ID)
		defer rows_to_process.Close()
		runningSQL.LoadMore = false
	}

	runningSQL.ScrollTo += recordsProcessed

	heading := runningSQL.Heading
	if heading == "" {
		heading = "Result"
	}

	queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: heading})

	return

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new UserModel type which wraps a database connection pool.
type ServerModel struct {
	DB *bolt.DB
}

func (m *ServerModel) getTableName() []byte {
	return []byte("servers")
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *ServerModel) Insert(u *Server) (string, error) {
	var id string = uuid.NewString()
	u.ID = id
	err := m.Update(u, false)

	return id, err
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *ServerModel) Update(u *Server, clearCache bool) error {
	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		u.Name = strings.ToUpper(strings.TrimSpace(u.Name))

		if !u.OnHold {
			u.OnHoldMessage = ""
		} else {
			go u.ClearCache()
		}

		if clearCache {
			go u.ClearCache()
		}

		buf, err := json.Marshal(u)
		if err != nil {
			return err
		}

		// key = > user.name+ user.id
		key := strings.ToUpper(u.ID) // + string(itob(u.ID))

		return bucket.Put([]byte(key), buf)
	})

	return err

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *ServerModel) Delete(id string) error {

	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		key := strings.ToUpper(id)
		dbDeleteError := bucket.Delete([]byte(key))
		return dbDeleteError
	})

	return err
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ServerModel) Exists(id string) bool {

	var userJson []byte

	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		key := strings.ToUpper(id)

		userJson = bucket.Get([]byte(key))

		return nil

	})

	return (userJson != nil)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ServerModel) DuplicateName(serverToCheck *Server) bool {
	exists := false
	for _, server := range m.List() {
		fmt.Println(">>>>duplucate name<<<", server.Name, "<>", serverToCheck.Name, "||", server.ID, "<>", serverToCheck.ID)
		if strings.EqualFold(server.Name, serverToCheck.Name) && !strings.EqualFold(server.ID, serverToCheck.ID) {
			exists = true
			break
		}
	}

	return exists
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ServerModel) Get(id string) (*Server, error) {

	if id == "" {
		return nil, errors.New("Server blank id not allowed")
	}
	var serverJSON []byte // = make([]byte, 0)

	err := m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableName())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		serverJSON = bucket.Get([]byte(strings.ToUpper(id)))

		return nil

	})
	server := Server{}
	if err != nil {
		return &server, err
	}

	// log.Println("serverJSON >2 >>", serverJSON)

	if serverJSON != nil {
		err := json.Unmarshal(serverJSON, &server)
		return &server, err

	}

	return &server, ErrServerNotFound

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ServerModel) List() []*Server {
	servers := make([]*Server, 0)
	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableName())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {

			server := Server{}
			err := json.Unmarshal(v, &server)
			if err == nil {
				servers = append(servers, &server)
			}
		}

		return nil
	})
	return servers

}
