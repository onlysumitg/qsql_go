package models

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/alexbrainman/odbc"
	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/database"
)

type SPParamter struct {
	Position           int
	Mode               string
	Name               string
	Datatype           string
	Scale              int
	Precision          int
	MaxLength          int
	DefaultValue       sql.NullString
	GlobalVariableName string
	CreateStatement    string
	DropStatement      string
	GivenValue         string
	OutValue           string
}

func (p *SPParamter) getDefaultValue() string {
	if p.DefaultValue.Valid {
		return p.DefaultValue.String
	}
	return ""
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s *Server) CallSP(runningSQL *RunningSql) (queryResults []*QueryResult) {
	notvalidErr := s.IsValid(runningSQL)
	if notvalidErr != nil {
		queryResults = append(queryResults, &QueryResult{Heading: "Onhold", ErrorMessage: notvalidErr.Error()})

		return
	}

	callID := strings.ReplaceAll(uuid.NewString(), "-", "")

	var re = regexp.MustCompile(`(?m)call\s*(.*)(\(.*\))`)
	match := re.FindStringSubmatch(runningSQL.RunningNow)

	brokenStatement := make([]string, 0)

	for i, _ := range re.SubexpNames() { //	for i, name := range re.SubexpNames() {
		if i > 0 && i <= len(match) {

			brokenStatement = append(brokenStatement, match[i])
		}
	}

	if len(brokenStatement) < 2 {
		queryResults = append(queryResults, &QueryResult{Heading: "Error", ErrorMessage: "Can not parse call statement"})
		return
	}
	spName, spLib := s.ParseSPName(brokenStatement[0])

	if spName == "" || spLib == "" {
		queryResults = append(queryResults, &QueryResult{Heading: "Error", ErrorMessage: "Can not parse parameters"})

		return
	}

	spParamters, err := s.GetSPParameter(spName, spLib)
	if err != nil {
		queryResults = append(queryResults, &QueryResult{Heading: "Error", ErrorMessage: err.Error()})

		return
	}

	givenParamterValues := s.ParseSPParamtersValues(brokenStatement[1])

	mapParamterToValues(spParamters, givenParamterValues)

	s.getGlobalVariables(spParamters, spName, spLib, callID)
	s.buildCreateGlobalVarSQLs(spParamters, spName, spLib)
	s.buildDropGlobalVarSQLs(spParamters, spName, spLib)
	callStatement, err := s.buildSpCallStatement(spName, spLib, spParamters, true)

	resultSets := s.GetSPResultSets(spName, spLib)

	if err != nil {
		return
	}

	fmt.Println(">>callStatement>>>>> >>>", callStatement)

	db, err := s.GetSinglaConnection()
	if err != nil {
		return
	}

	// create global variable
	for _, spParamter := range spParamters {
		if spParamter.CreateStatement != "" {
			_, err := db.Exec(spParamter.CreateStatement)
			if err != nil {
				var odbcError *odbc.Error

				if errors.As(err, &odbcError) {
					s.UpdateAfterError(odbcError)
				}
				log.Printf(spParamter.CreateStatement, err.Error())
			}
		}
	}

	if resultSets == 0 {
		// call sp
		_, err = db.Exec(callStatement) //"select * from sumitg1/qsqltest")'
		if err != nil {
			var odbcError *odbc.Error

			if errors.As(err, &odbcError) {
				s.UpdateAfterError(odbcError)
			}
			queryResults = append(queryResults, &QueryResult{Heading: "Error", ErrorMessage: err.Error()})

			return
		}
	} else {
		rows, err := db.Query(callStatement)
		if err != nil {
			var odbcError *odbc.Error

			if errors.As(err, &odbcError) {
				s.UpdateAfterError(odbcError)
			}
			queryResults = append(queryResults, &QueryResult{Heading: "Error", ErrorMessage: err.Error()})

			return
		}
		resultset_data, resultset_columns := database.ToMap(rows, -1, runningSQL.ScrollTo)

		resultsetCounter := 1
		queryResults = append(queryResults, &QueryResult{Rows: resultset_data, Columns: resultset_columns, Heading: fmt.Sprintf("%s Resultset %d", spName, resultsetCounter)})

		for rows.NextResultSet() {
			resultsetCounter += 1
			resultset_data, resultset_columns = database.ToMap(rows, -1, runningSQL.ScrollTo)
			queryResults = append(queryResults, &QueryResult{Rows: resultset_data,
				Columns: resultset_columns, Heading: fmt.Sprintf("%s Resultset %d", spName, resultsetCounter)})

		}
	}

	// need to process rows for result set

	// get values from global variables
	globalVariableNames := make([]string, 0)
	for _, spParamter := range spParamters {
		if spParamter.GlobalVariableName != "" {
			globalVariableNames = append(globalVariableNames, fmt.Sprintf("%s as %s", spParamter.GlobalVariableName, spParamter.Name))

		}
	}
	if len(globalVariableNames) > 0 {

		sql := fmt.Sprintf("select %s  from sysibm/SYSDUMMY1", strings.Join(globalVariableNames, ","))
		rows, err_query := db.Query(sql)
		if err_query != nil {
			var odbcError *odbc.Error

			if errors.As(err, &odbcError) {
				s.UpdateAfterError(odbcError)
			}
			err = err_query
			return
		}
		return_parameters, paramter_types := database.ToMap(rows, len(globalVariableNames), runningSQL.ScrollTo)
		queryResults = append(queryResults, &QueryResult{Rows: return_parameters, Columns: paramter_types, Heading: fmt.Sprintf("%s Out paramters", spName)})

	}
	// drop global variables
	for _, spParamter := range spParamters {
		if spParamter.DropStatement != "" {
			conn, _ := s.GetConnection()
			_, err := conn.Exec(spParamter.DropStatement)
			if err != nil {
				var odbcError *odbc.Error

				if errors.As(err, &odbcError) {
					s.UpdateAfterError(odbcError)
				}
				log.Printf(spParamter.CreateStatement, err.Error())
			}
		}
	}

	return

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) ParseSPName(spQualifiedName string) (spName string, spLib string) {

	seperator := "/"
	if strings.Contains(spQualifiedName, ".") {
		seperator = "."
	}
	spNameBroken := strings.Split(spQualifiedName, seperator)
	spName = spNameBroken[1]
	spLib = spNameBroken[0]

	return
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) ParseSPParamtersValues(paramString string) map[string]string {

	paramString2 := strings.TrimPrefix(paramString, "(")

	paramString2 = strings.TrimSuffix(paramString2, ")")
	paramMap := make(map[string]string)
	parser := csv.NewReader(strings.NewReader(paramString2))
	parser.Comma = ','
	parser.LazyQuotes = true
	csvLines, err := parser.ReadAll()
	if err != nil {
		fmt.Println(">>>> csv err line>>>> ", err)
	}
	paramCounter := 0
	for _, line := range csvLines {
		for _, param := range line {
			paramCounter += 1
			if strings.Contains(param, "=>") && !strings.HasPrefix(param, "=>") && !strings.HasSuffix(param, "=>") {
				temp := strings.Split(param, "=>")
				paramMap[strings.ToUpper(temp[0])] = temp[1]

			} else {
				paramMap[fmt.Sprint(paramCounter)] = param
			}

		}
	}

	return paramMap
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) GetSPResultSets(spName, spLib string) int {

	resultSets := 0

	sql := fmt.Sprintf("select RESULT_SETS from qsys2.sysprocs where SPECIFIC_NAME='%s'  and SPECIFIC_SCHEMA='%s'", strings.ToUpper(spName), strings.ToUpper(spLib))
	conn, err := s.GetConnection()

	if err != nil {
		log.Println("GetSPResultSets 1 ", err.Error())
		return 0
	}
	row := conn.QueryRow(sql)

	err = row.Scan(&resultSets)

	if err != nil {
		log.Println("GetSPResultSets 2 ", err.Error())

	}

	return resultSets
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) GetSPParameter(spName, spLib string) (params []*SPParamter, err error) {
	sql := fmt.Sprintf("SELECT ORDINAL_POSITION, trim(PARAMETER_MODE) , PARAMETER_NAME,DATA_TYPE, ifnull(NUMERIC_SCALE,0), ifnull(NUMERIC_PRECISION,0), ifnull(CHARACTER_MAXIMUM_LENGTH,0),  default FROM qsys2.sysparms WHERE SPECIFIC_NAME='%s' and   SPECIFIC_SCHEMA ='%s' ORDER BY ORDINAL_POSITION", strings.ToUpper(spName), strings.ToUpper(spLib))

	conn, err := s.GetConnection()

	if err != nil {

		return
	}

	rows, err := conn.Query(sql)
	if err != nil {
		var odbcError *odbc.Error

		if errors.As(err, &odbcError) {
			s.UpdateAfterError(odbcError)
		}
		return
	}

	for rows.Next() {
		spParamter := &SPParamter{}
		err := rows.Scan(&spParamter.Position,
			&spParamter.Mode,
			&spParamter.Name,
			&spParamter.Datatype,
			&spParamter.Scale,
			&spParamter.Precision,
			&spParamter.MaxLength,
			&spParamter.DefaultValue)
		if err != nil {
			log.Println("GetSPParameter ", err.Error())
		}

		params = append(params, spParamter)

	}

	return

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) buildSpCallStatement(spName, spLib string, parameters []*SPParamter, useNamedParams bool) (callStatement string, err error) {

	paramString := ""

	for _, parameter := range parameters {
		value := ""
		switch parameter.Mode {
		case "IN":
			value = fmt.Sprintf("'%s'", parameter.GivenValue)
		case "OUT":
			value = parameter.GlobalVariableName
		case "INOUT":
			value = parameter.GlobalVariableName
		}

		if useNamedParams {
			paramString += fmt.Sprintf("%s=>%s %s", parameter.Name, value, ",")
		} else {
			paramString += fmt.Sprintf("%s %s", value, ",")
		}

	}

	paramString = strings.TrimRight(paramString, ",")
	callStatement = fmt.Sprintf("call %s.%s(%s)", spLib, spName, paramString)
	return
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) getGlobalVariables(parameters []*SPParamter, spName, spLib string, callId string) {
	for _, spParamter := range parameters {
		if spParamter.Mode == "OUT" || spParamter.Mode == "INOUT" {
			spParamter.GlobalVariableName = fmt.Sprintf("%s.%s_%s_%s", s.WorkLib, spName, callId, spParamter.Name)
		}
	}
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) buildDropGlobalVarSQLs(parameters []*SPParamter, spName, spLib string) {
	for _, spParamter := range parameters {
		if spParamter.Mode == "OUT" || spParamter.Mode == "INOUT" {
			spParamter.DropStatement = s.buildDropGlobalVarSQL(spParamter, spName, spLib)
		}
	}
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) buildCreateGlobalVarSQLs(parameters []*SPParamter, spName, spLib string) {
	for _, spParamter := range parameters {
		if spParamter.Mode == "OUT" || spParamter.Mode == "INOUT" {
			spParamter.CreateStatement = s.buildCreateGlobalVarSQL(spParamter, spName, spLib)
		}
	}
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) buildCreateGlobalVarSQL(parameter *SPParamter, spName, spLib string) (sqlStatement string) {
	defaultValue := parameter.getDefaultValue()
	switch parameter.Datatype {
	case "NUMERIC":
		if defaultValue == "" {
			defaultValue = "0"
		}
		sqlStatement = fmt.Sprintf("create or replace variable %s %s(%d,%d) default %s",
			parameter.GlobalVariableName, parameter.Datatype, parameter.Precision, parameter.Scale, defaultValue)

	default:
		if defaultValue == "" {
			defaultValue = "''"
		}
		if parameter.GivenValue != "" {
			defaultValue = fmt.Sprintf("'%s'", parameter.GivenValue)
		}
		sqlStatement = fmt.Sprintf("create or replace variable %s %s(%d) default %s",
			parameter.GlobalVariableName, parameter.Datatype, parameter.MaxLength, defaultValue)
	}
	return

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s Server) buildDropGlobalVarSQL(parameter *SPParamter, spName, spLib string) (sqlStatement string) {
	sqlStatement = fmt.Sprintf("drop variable %s", parameter.GlobalVariableName)
	return
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func mapParamterToValues(params []*SPParamter, givenParamValueMap map[string]string) {
	for index, paramter := range params {
		givenValue, found := givenParamValueMap[paramter.Name]
		if !found {
			givenValue = givenParamValueMap[fmt.Sprint(index+1)]
		}

		paramter.GivenValue = givenValue

	}
}
