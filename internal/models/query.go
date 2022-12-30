package models

import (
	"encoding/csv"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/zerobit-tech/godbc/database/sql"

	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/database"
)

type RunningSql struct {
	ID            string
	Sql           string
	RunningNow    string
	StatementType string
	LoadMore      bool
	ScrollTo      int
	ResultSetSize int
	LimitRecods   bool
	Heading       string
}

var runningSQLQueryMap map[string]*sql.Rows = make(map[string]*sql.Rows, 10)

type QueryResult struct {
	Heading      string
	CurrentSql   RunningSql
	Rows         []map[string]interface{}
	Columns      []database.ColumnType
	FlashMessage string
	ErrorMessage string
}

// func prepareSelectStatement(runningSQL *RunningSql, server Server) {
// 	var re = regexp.MustCompile(`(?mi)(limit\s*\d*\s*)|(offset\s*\d*\s*)|(fetch\s*first\s*\d\s*(row|rows)\s*only)$`)
// 	finalSQL := runningSQL.Sql

// 	defaultLimit := int(server.GetDefaultLimit())

// 	if !re.MatchString(finalSQL) {
// 		finalSQL = finalSQL + " limit " + strconv.Itoa(defaultLimit) + " offset " + strconv.Itoa(runningSQL.Offset)

// 		runningSQL.Offset = runningSQL.Offset + defaultLimit
// 		runningSQL.LoadMore = true
// 	} else {
// 		runningSQL.Offset = 0
// 	}

// 	runningSQL.RunningNow = finalSQL

// }
// ------------------------------------------------------
//
// ------------------------------------------------------
func PrepareSQLToRun(runningSQL *RunningSql) {
	finalSQL := strings.Trim(runningSQL.Sql, " ")
	finalSQL = strings.ToUpper(finalSQL)
	runningSQL.LimitRecods = true
	runningSQL.ResultSetSize = 10

	switch {

	case strings.HasPrefix(finalSQL, "COMMIT"):
		runningSQL.StatementType = "COMMIT"
		runningSQL.RunningNow = runningSQL.Sql

	case strings.HasPrefix(finalSQL, "ROLLBACK"):
		runningSQL.StatementType = "ROLLBACK"
		runningSQL.RunningNow = runningSQL.Sql

	case strings.HasPrefix(finalSQL, "INSERT"):
		runningSQL.StatementType = "INSERT"
		runningSQL.RunningNow = runningSQL.Sql

	case strings.HasPrefix(finalSQL, "UPDATE"):
		runningSQL.StatementType = "UPDATE"
		runningSQL.RunningNow = runningSQL.Sql

	case strings.HasPrefix(finalSQL, "DELETE"):
		runningSQL.StatementType = "DELETE"
		runningSQL.RunningNow = runningSQL.Sql

	case strings.HasPrefix(finalSQL, "CALL"):
		runningSQL.StatementType = "CALL"
		runningSQL.RunningNow = runningSQL.Sql

	case strings.HasPrefix(finalSQL, "SELECT") || strings.HasPrefix(finalSQL, "WITH"):
		runningSQL.StatementType = "SELECT"
		//prepareSelectStatement(runningSQL, server)
		runningSQL.RunningNow = runningSQL.Sql

	case strings.HasPrefix(finalSQL, "@HEADING"):
		var re = regexp.MustCompile(`(?mi)(@heading:.+?){1} `)
		matches := re.FindStringSubmatch(runningSQL.Sql)

		if len(matches) >= 2 {
			headings := strings.Split(matches[1], ":")
			runningSQL.Heading = headings[1]
			runningSQL.RunningNow = re.ReplaceAllString(runningSQL.Sql, "")

		}

		runningSQL.Sql = runningSQL.RunningNow // 2nd time it will work for actual sql type

		PrepareSQLToRun(runningSQL)

	case strings.HasPrefix(finalSQL, "@BATCH"):
		re := regexp.MustCompile(`(?i)@BATCH`)
		runningSQL.StatementType = "@BATCH"
		runningSQL.RunningNow = re.ReplaceAllString(runningSQL.Sql, "")
		runningSQL.ResultSetSize = 5000
		runningSQL.LimitRecods = false
		runningSQL.Sql = runningSQL.RunningNow // 2nd time it will work for actual sql type

	default:

		if strings.HasPrefix(finalSQL, "@") {
			for key, value := range QueryMap {
				if strings.EqualFold(key, finalSQL) {
					runningSQL.Sql = value
					break
				}
			}
		}

		runningSQL.StatementType = "OTHER"
		runningSQL.RunningNow = runningSQL.Sql
	}

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func ActuallyRunSQL(server *Server, runningSQL RunningSql, ch chan<- []*QueryResult, wg *sync.WaitGroup) {
	queryResult := ActuallyRunSQL2(server, runningSQL)
	wg.Done()
	ch <- queryResult
}

// ------------------------------------------------------
//
// ------------------------------------------------------
func ActuallyRunSQL2(server *Server, runningSQL RunningSql) []*QueryResult {

	QueryResults := server.RunQuery(&runningSQL)
	return QueryResults
}

// ------------------------------------------------------
//
// ------------------------------------------------------
func ProcessSQLStatements(sqlStatements string, server *Server) []QueryResult {
	sqlsToProcess := PrepareSQLStatements(sqlStatements, *server)

	var queryResults []QueryResult
	if len(sqlsToProcess) > 1 {
		queryResults = runMultpleSQLs(sqlsToProcess, server)
	} else {
		log.Println("Running as single SQL")
		//queryResults = runSingleSQL(sqlsToProcess[0], currentServer)
		queryResults = runMultpleSQLs(sqlsToProcess, server)

	}

	return queryResults

}

// ------------------------------------------------------
//
// ------------------------------------------------------
func runSingleSQL(sqlsToProcess RunningSql, currentServer *Server) []QueryResult {
	queryResults := make([]QueryResult, 0)

	PrepareSQLToRun(&sqlsToProcess)

	queryResultList := ActuallyRunSQL2(currentServer, sqlsToProcess)
	for _, queryResult := range queryResultList {
		queryResults = append(queryResults, *queryResult)
	}
	return queryResults
}

// ------------------------------------------------------
//
// ------------------------------------------------------
func runMultpleSQLs(sqlsToProcess []RunningSql, currentServer *Server) []QueryResult {
	queryResults := make([]QueryResult, 0)

	var wg sync.WaitGroup

	var chList []chan []*QueryResult

	for _, runningSQL := range sqlsToProcess {
		if runningSQL.RunningNow != "" {
			ch := make(chan []*QueryResult)
			chList = append(chList, ch)
			wg.Add(1)
			go ActuallyRunSQL(currentServer, runningSQL, ch, &wg)
		}

	}
	wg.Wait()

	for _, ch := range chList {
		queryResultList := <-ch
		for _, queryResult := range queryResultList {
			queryResults = append(queryResults, *queryResult)
		}
		close(ch)

	}

	return queryResults
}

// ------------------------------------------------------
//
// ------------------------------------------------------
func PrepareSQLStatements(sqlStatements string, server Server) []RunningSql {

	var responseLines []RunningSql = make([]RunningSql, 0)
	sqlStatements = strings.Replace(sqlStatements, "\n", " ", -1)

	parser := csv.NewReader(strings.NewReader(sqlStatements))
	parser.Comma = ';'
	parser.LazyQuotes = true
	csvLines, err := parser.ReadAll()
	if err != nil {
		fmt.Println(">>>> csv err line>>>> ", err)
	}

	for _, line := range csvLines {
		for _, sql := range line {
			sql = strings.Trim(sql, " ")
			if sql != "" {
				// generate new running sql
				currentSql := &RunningSql{}
				currentSql.ID = uuid.NewString()
				currentSql.Sql = sql
				PrepareSQLToRun(currentSql)

				responseLines = append(responseLines, *currentSql)
			}
		}
	}

	return responseLines

}
