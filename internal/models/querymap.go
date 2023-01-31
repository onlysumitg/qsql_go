package models

var QueryMap = make(map[string]string)

func LoadQueryMap(shorthandQueries *ShorthandQueryModel, savedQueries *SavedQueryModel) {
	QueryMap = make(map[string]string)
	QueryMap["@LIBL"] = "SELECT * FROM QSYS2.LIBRARY_LIST_INFO"
	QueryMap["@JOB"] = "SELECT JOB_NAME,AUTHORIZATION_NAME,JOB_TYPE,JOB_STATUS,SUBSYSTEM,SUBSYSTEM_LIBRARY_NAME FROM TABLE(QSYS2.ACTIVE_JOB_INFO(JOB_NAME_FILTER => '*')) A ORDER BY ELAPSED_CPU_PERCENTAGE DESC"
	QueryMap["@MSGW"] = "SELECT JOB_NAME,JOB_TYPE,JOB_STATUS,SUBSYSTEM FROM TABLE(QSYS2.ACTIVE_JOB_INFO()) B WHERE JOB_STATUS = 'MSGW' ORDER BY JOB_NAME"

	for key, value := range QueryMap {
		if !shorthandQueries.NameExists(key) {
			shorthandQuery := ShorthandQuery{Name: key, Sql: value}
			shorthandQueries.Save(&shorthandQuery)
		}
	}
	AutoGenerateForSavedQueries(savedQueries)

	for _, queryAlias := range shorthandQueries.List() {
		QueryMap[queryAlias.Name] = queryAlias.Sql

	}

}

func ReloadQueryMap(shorthandQueries *ShorthandQueryModel) {
	for _, queryAlias := range shorthandQueries.List() {
		QueryMap[queryAlias.Name] = queryAlias.Sql

	}

}
