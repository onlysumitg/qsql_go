package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zerobit-tech/godbc/database/sql"

	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/database"
	"github.com/onlysumitg/qsql2/internal/validator"
	bolt "go.etcd.io/bbolt"
)

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new User type. Notice how the field names and types align
// with the columns in the database "users" table?
type Server struct {
	ID          string    `json:"id" db:"id" form:"id"`
	Name        string    `json:"server_name" db:"server_name" form:"name"`
	IP          string    `json:"ip" db:"ip" form:"ip"`
	Port        uint8     `json:"port" db:"port" form:"port"`
	Ssl         bool      `json:"ssl" db:"ssl" form:"ssl"`
	UserName    string    `json:"user_name" db:"user_name" form:"user_name"`
	Password    string    `json:"password" db:"password" form:"password"`
	WorkLib     string    `json:"worklib" db:"worklib" form:"worklib"`
	CreatedAt   time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at" form:"-"`
	Connections uint      `json:"connections" db:"connections" form:"connections"`
	validator.Validator
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) GetConnectionString() string {
	driver := "IBM i Access ODBC Driver"
	connectionString := fmt.Sprintf("DRIVER=%s;SYSTEM=%s; UID=%s;PWD=%s;DBQ=*USRLIBL;UNICODESQL=1;XDYNAMIC=1;EXTCOLINFO=1;PKG=A/DJANGO,2,0,0,1,512;PROTOCOL=TCPIP;NAM=1;CMT=0;", driver, s.IP, s.UserName, s.Password)
	return connectionString
}

func (s Server) GetDefaultLimit() uint {
	defaultLimit := s.Connections
	if defaultLimit <= 0 {
		defaultLimit = 10
	}
	return defaultLimit
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
func (s Server) GetConnection() (*sql.DB, error) {
	return database.GetConnection(s)
}
func (s Server) GetSinglaConnection() (*sql.DB, error) {
	return database.GetSingleConnection(s)
}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) RunQuery(runningSQL *RunningSql) (queryResults []*QueryResult) {

	switch runningSQL.StatementType {
	case "INSERT":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "UPDATE":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "DELETE":
		queryResults = s.RunExecuteQuery(runningSQL)
	case "CALL":
		queryResults = s.CallSP(runningSQL)
	case "@BATCH":
		queryResults = s.BatchStatement(runningSQL)

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
func (s Server) BatchStatement(runningSQL *RunningSql) (queryResults []*QueryResult) {
	return_rows := make([]map[string]interface{}, 0)
	column_types := make([]database.ColumnType, 0)
	runningSQL.LoadMore = false

	queryResult := &QueryResult{CurrentSql: *runningSQL,
		Rows: return_rows, Columns: column_types,
		Heading:      "Result",
		FlashMessage: "Batch job has been submitted",
	}

	queryResults = append(queryResults, queryResult)
	//batchSql.
	return

}

// ------------------------------------------------------------
//
// ------------------------------------------------------------
func (s Server) RunExecuteQuery(runningSQL *RunningSql) (queryResults []*QueryResult) {
	return_rows := make([]map[string]interface{}, 0)
	column_types := make([]database.ColumnType, 0)

	conn, err_connection := s.GetConnection()

	if err_connection != nil {
		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Connection Error", ErrorMessage: err_connection.Error()})

		return
	}

	res, err_query := conn.Exec(runningSQL.RunningNow) //"select * from sumitg1/qsqltest")
	if err_query != nil {
		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Query Error", ErrorMessage: err_query.Error()})
		return
	}

	count, err2 := res.RowsAffected()
	if err2 != nil {
		queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Error", ErrorMessage: err2.Error()})
		return
	}
	runningSQL.LoadMore = false
	fmt.Println(count)
	// return_rows, column_types = database.ToMap(rows_to_process, 10, runningSQL.ScrollTo)
	column_type := database.ColumnType{
		Name: "Rows Impacted",
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
func (s Server) RunSelectQuery(runningSQL *RunningSql) (queryResults []*QueryResult) {
	return_rows := make([]map[string]interface{}, 0)
	column_types := make([]database.ColumnType, 0)

	rows_to_process, ok := runningSQLQueryMap[runningSQL.ID]

	if !ok {
		conn, err_connection := s.GetConnection()

		if err_connection != nil {
			queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Connection Error", ErrorMessage: err_connection.Error()})
			return
		}
 		rows, err_query := conn.Query(runningSQL.RunningNow) //"select * from sumitg1/qsqltest")
		if err_query != nil {
			queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Query Error", ErrorMessage: err_query.Error()})
			return
		}

		rows_to_process = rows

	}

	return_rows, column_types = database.ToMap(rows_to_process, 10, runningSQL.ScrollTo)

	recordsProcessed := len(return_rows)
	if recordsProcessed >= 10 {
		runningSQL.LoadMore = true
		runningSQLQueryMap[runningSQL.ID] = rows_to_process
	}

	if runningSQL.LoadMore && len(return_rows) < int(s.GetDefaultLimit()) {
		runningSQL.LoadMore = false
	}

	runningSQL.ScrollTo += recordsProcessed

	queryResults = append(queryResults, &QueryResult{Rows: return_rows, Columns: column_types, Heading: "Result"})

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
	err := m.Update(u)

	return id, err
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *ServerModel) Update(u *Server) error {
	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		u.Name = strings.ToUpper(strings.TrimSpace(u.Name))

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
		return nil, errors.New("blank id not allowed")
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
