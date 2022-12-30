package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/batch"
	"github.com/onlysumitg/qsql2/internal/database"
	"github.com/onlysumitg/qsql2/internal/iwebsocket"
	"github.com/onlysumitg/qsql2/internal/validator"
	bolt "go.etcd.io/bbolt"
)

type BatchSqlForm struct {
	ID                  string `json:"id" db:"id" form:"id"`
	Sql                 string `json:"sql" db:"sql" form:"sql"`
	RepeatEvery         int    `json:"repeatevery" db:"repeatevery" form:"repeatevery"`
	RepeatXtimes        int    `json:"repeatxtimes" db:"repeatxtimes" form:"repeatxtimes"`
	validator.Validator        // this contains the fielderror

}
type BatchSql struct {
	Server
	RunningSql
	Status string

	RepeatEvery     time.Duration `json:"repeatevery" db:"repeatevery" form:"repeatevery"`
	RepeatXtimes    int           `json:"repeatxtimes" db:"repeatxtimes" form:"repeatxtimes"`
	NextRun         time.Time
	ProcessedXtimes int

	CreatedAt   time.Time `json:"created_at" db:"created_at" form:"-"`
	CompletedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

type BatchSQLRun struct {
	ID string

	ParentId   string
	RunCounter int
	Notified   bool

	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`

	QueryResults []QueryResult
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (m *BatchSql) IsPending() bool {
	if m.RepeatXtimes > 0 && m.ProcessedXtimes >= m.RepeatXtimes {
		return false
	}

	return true
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (m *BatchSql) IsReadyToRun() bool {
	if !m.IsPending() {
		return false
	}
	if m.NextRun.IsZero() {
		m.NextRun = m.CompletedAt.Add(m.RepeatEvery)
	}

	if m.NextRun.Equal(time.Now()) || m.NextRun.Before(time.Now()) {
		return true
	}

	return false
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------

func (m *BatchSql) SetNextRunTime() {
	if m.NextRun.IsZero() {
		m.NextRun = m.CompletedAt.Add(m.RepeatEvery)
	}

	m.NextRun = time.Now().Add(m.RepeatEvery)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new UserModel type which wraps a database connection pool.
type BatchSQLModel struct {
	DB *bolt.DB
}

func (m *BatchSQLModel) getTableName() []byte {
	return []byte("batchsqls3")
}

func (m *BatchSQLModel) getTableNameForRun() []byte {
	return []byte("batchsqlrun")
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (m *BatchSQLModel) BatchProcess() *batch.BatchProcess {
	return &batch.BatchProcess{
		BatchFunction: m.ProcessPending,
		RunEvery:      time.Second * 30,
	}
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (m *BatchSQLModel) ProcessPending(mutex *sync.Mutex) {
	log.Println("Starting process pending Batch SQLs")
	mutex.Lock()
	defer mutex.Unlock()

	var wg sync.WaitGroup

	for _, batchSQL := range m.List() {
		if batchSQL.IsReadyToRun() {
			wg.Add(1)
			go m.ProcessSingle(batchSQL, &wg)

		}
	}

	wg.Wait()

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (m *BatchSQLModel) ProcessSingle(batchSQL *BatchSql, wg *sync.WaitGroup) {
	defer wg.Done()
	batchSQL.ProcessedXtimes += 1

	batchSQLRun := &BatchSQLRun{ParentId: batchSQL.RunningSql.ID,
		RunCounter: batchSQL.ProcessedXtimes,
	}

	batchSQLRun.QueryResults = make([]QueryResult, 0)
	return_rows := make([]map[string]interface{}, 0)
	column_types := make([]database.ColumnType, 0)

	serverModel := &ServerModel{DB: m.DB}

	serverToUse, err := serverModel.Get(batchSQL.Server.ID)
	if err != nil {
		batchSQLRun.QueryResults = append(batchSQLRun.QueryResults, QueryResult{Rows: return_rows, Columns: column_types, Heading: "Error", ErrorMessage: err.Error()})

		m.Update(batchSQL)
		m.SaveRun(batchSQLRun)
		return
	}

	if serverToUse.OnHold {
		batchSQLRun.QueryResults = append(batchSQLRun.QueryResults, QueryResult{Rows: return_rows, Columns: column_types, Heading: "Error", ErrorMessage: "Server is on hold"})
		m.Update(batchSQL)
		m.SaveRun(batchSQLRun)

		return
	}

	batchSQL.Status = "STARTED"

	originalResultSetSize := batchSQL.RunningSql.ResultSetSize
	originalLimitRecods := batchSQL.RunningSql.LimitRecods

	PrepareSQLToRun(&batchSQL.RunningSql)

	batchSQL.RunningSql.ResultSetSize = originalResultSetSize
	batchSQL.RunningSql.LimitRecods = originalLimitRecods

	// fmt.Println("runningSQL.ResultSetSize >>>>>>>> ?>>>>>>>", batchSQL.RunningSql.ResultSetSize)
	queryResults := ActuallyRunSQL2(serverToUse, batchSQL.RunningSql)
	serverModel.Update(serverToUse, false)

	for _, qr := range queryResults {
		batchSQLRun.QueryResults = append(batchSQLRun.QueryResults, *qr)
	}

	iwebsocket.BroadcastNotification(fmt.Sprintf("Batch query %s completed.", batchSQL.RunningSql.ID), "success")

	batchSQL.CompletedAt = time.Now()

	if batchSQL.RepeatXtimes > batchSQL.ProcessedXtimes {
		batchSQL.Status = fmt.Sprintf("Processed %d/%d", batchSQL.ProcessedXtimes, batchSQL.RepeatXtimes)
	} else {
		batchSQL.Status = "COMPLETED"

	}

	batchSQL.SetNextRunTime()
	m.Update(batchSQL)
	m.SaveRun(batchSQLRun)

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *BatchSQLModel) Insert(u *BatchSql) (string, error) {
	var id string = uuid.NewString()
	u.RunningSql.ID = id
	u.Status = "PENDING"
	u.CreatedAt = time.Now()
	err := m.Update(u)

	return id, err
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *BatchSQLModel) Update(u *BatchSql) error {
	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}

		buf, err := json.Marshal(u)
		if err != nil {
			return err
		}

		// key = > user.name+ user.id
		key := strings.ToUpper(u.RunningSql.ID) // + string(itob(u.ID))

		return bucket.Put([]byte(key), buf)
	})

	return err

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *BatchSQLModel) List() []*BatchSql {
	batchSqls := make([]*BatchSql, 0)
	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableName())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {

			batchSql := BatchSql{}
			err := json.Unmarshal(v, &batchSql)
			if err == nil {
				batchSqls = append(batchSqls, &batchSql)
			}
		}

		return nil
	})
	return batchSqls

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *BatchSQLModel) Get(id string) (*BatchSql, error) {

	if id == "" {
		return nil, errors.New("BatchSql blank id not allowed")
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
	batchSql := BatchSql{}
	if err != nil {
		return &batchSql, err
	}

	// log.Println("serverJSON >2 >>", serverJSON)

	if serverJSON != nil {
		err := json.Unmarshal(serverJSON, &batchSql)
		return &batchSql, err

	}

	return &batchSql, ErrServerNotFound

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *BatchSQLModel) Delete(id string) error {
	batchSql, err := m.Get(id)
	if err == nil {
		for _, run := range m.ListRun(batchSql) {
			go m.DeleteRun(run.ID)
		}
	}
	err = m.DB.Update(func(tx *bolt.Tx) error {
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
// We'll use the Insert method to add a new record to the "users" table.
func (m *BatchSQLModel) ListRun(u *BatchSql) []*BatchSQLRun {
	id := strings.ToUpper(u.RunningSql.ID)

	batchSqlRuns := make([]*BatchSQLRun, 0)
	err := m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableNameForRun())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		c := bucket.Cursor()
		fmt.Println("Looking for prefix.........", id)
		prefix := []byte(id)

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			//for k, v := c.First(); k != nil; k, v = c.Next() {

			batchSqlRun := BatchSQLRun{}
			err := json.Unmarshal(v, &batchSqlRun)
			if err == nil {
				batchSqlRuns = append(batchSqlRuns, &batchSqlRun)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println("Error ListRun::", err.Error())
	}

	return batchSqlRuns
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *BatchSQLModel) SaveRun(u *BatchSQLRun) error {
	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableNameForRun())
		if err != nil {
			return err
		}

		u.CreatedAt = time.Now()

		// key = > user.name+ user.id
		key := strings.ToUpper(u.ParentId) // + string(itob(u.ID))
		key = fmt.Sprintf("%s_%d", key, u.RunCounter)
		if u.ID == "" {
			u.ID = key
		}

		buf, err := json.Marshal(u)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(key), buf)
	})

	return err

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *BatchSQLModel) GetRun(id string) (*BatchSQLRun, error) {

	if id == "" {
		return nil, errors.New("BatchSQLRun blank id not allowed")
	}
	var serverJSON []byte // = make([]byte, 0)

	err := m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableNameForRun())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		serverJSON = bucket.Get([]byte(strings.ToUpper(id)))

		return nil

	})
	batchSqlRun := BatchSQLRun{}
	if err != nil {
		return &batchSqlRun, err
	}

	// log.Println("serverJSON >2 >>", serverJSON)

	if serverJSON != nil {
		err := json.Unmarshal(serverJSON, &batchSqlRun)
		return &batchSqlRun, err

	}

	return &batchSqlRun, ErrNotFound

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *BatchSQLModel) DeleteRun(id string) error {
	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableNameForRun())
		if err != nil {
			return err
		}
		key := strings.ToUpper(id)
		dbDeleteError := bucket.Delete([]byte(key))
		return dbDeleteError
	})

	return err
}
