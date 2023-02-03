package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/alexedwards/scs/boltstore"
	"github.com/alexedwards/scs/v2"
	bolt "go.etcd.io/bbolt"
)

func getSessionManager(db *bolt.DB) *scs.SessionManager {
	// Use the scs.New() function to initialize a new session manager. Then we
	// configure it to use our MySQL database as the session store, and set a
	// lifetime of 12 hours (so that sessions automatically expire 12 hours
	// after first being created).
	fmt.Printf("Setting session")

	sessionManager := scs.New()
	sessionManager.Lifetime = 48 * time.Hour
	sessionManager.Store = boltstore.NewWithCleanupInterval(db, 200*time.Second)
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	//sessionManager.Cookie.Secure = app.InProduction

	// 	sessionManager.Lifetime = 3 * time.Hour
	// sessionManager.IdleTimeout = 20 * time.Minute
	// sessionManager.Cookie.Name = "session_id"
	// sessionManager.Cookie.Domain = "example.com"
	// sessionManager.Cookie.HttpOnly = true
	// sessionManager.Cookie.Path = "/example/"
	// sessionManager.Cookie.Persist = true
	// sessionManager.Cookie.SameSite = http.SameSiteStrictMode
	// sessionManager.Cookie.Secure = true

	return sessionManager
}
