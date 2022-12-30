package main

import (
	"html/template"
	"log"
	"os"

	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form"
	"github.com/onlysumitg/qsql2/internal/models"

	bolt "go.etcd.io/bbolt"
)

type application struct {
	errorLog         *log.Logger
	infoLog          *log.Logger
	templateCache    map[string]*template.Template
	formDecoder      *form.Decoder
	sessionManager   *scs.SessionManager
	users            *models.UserModel
	servers          *models.ServerModel
	savedQueries     *models.SavedQueryModel
	shorthandQueries *models.ShorthandQueryModel

	batchSQLModel *models.BatchSQLModel
	InProduction  bool
	hostURL       string
}

func baseAppConfig(params parameters, db *bolt.DB) *application {
	//--------------------------------------- Setup loggers ----------------------------
	infoLog := log.New(os.Stderr, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	//--------------------------------------- Setup template cache ----------------------------
	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}
	//--------------------------------------- Setup form decoder ----------------------------
	formDecoder := form.NewDecoder()

	//---------------------------------------  final app config ----------------------------
	app := &application{
		errorLog:         errorLog,
		infoLog:          infoLog,
		templateCache:    templateCache,
		sessionManager:   getSessionManager(db),
		formDecoder:      formDecoder,
		users:            &models.UserModel{DB: db},
		servers:          &models.ServerModel{DB: db},
		savedQueries:     &models.SavedQueryModel{DB: db},
		shorthandQueries: &models.ShorthandQueryModel{DB: db},
		batchSQLModel:    &models.BatchSQLModel{DB: db},
		hostURL:          params.getHttpAddress(),
	}

	return app

}
