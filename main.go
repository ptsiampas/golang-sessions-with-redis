package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sessions-with-redis/session"
	"time"
)

var ses session.Session

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

var users = map[string]string{
	"user1": "password1",
	"user2": "password2",
}


func main() {
	ses.DefaultSessionTimeout = 120

	// Signin and "Welcome" are the handlers
	http.HandleFunc("/signin", Signin)
	http.HandleFunc("/welcome", Welcome)
	http.HandleFunc("/refresh", Refresh)

	log.Fatalln(http.ListenAndServe("0.0.0.0:8080", nil))
}

func Signin(w http.ResponseWriter, r *http.Request) {
	var creds Credentials


	// Get the JSON Body and decode into credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err !=nil {
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the expected password from our in memory map
	expectedPassword, ok := users[creds.Username]
	// If a password exists for the given user
	// AND, if it is the same as the password we received, the we can move ahead
	// if NOT, then we return an "Unauthorized" status
	if !ok || expectedPassword != creds.Password {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sessionToken, err := ses.Create(creds.Username)
	webErrAndLog(w, err)

	// Finally, we set the client cookie for "session_token" as the session token we just generated
	// we also set an expiry time of 120 seconds, the same as the cache
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(time.Duration(ses.DefaultSessionTimeout) * time.Second),
	})
}

func Welcome(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return an unauthorized status
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// For any other type of error, return a bad request status
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	response, err := ses.Get(sessionToken)
	webErrAndLog(w, err)

	// Finally, return the welcome message to the user
	_, err = w.Write([]byte(fmt.Sprintf("Welcome %s!", response)))
	if err != nil {
		log.Println(err)
	}
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionToken := c.Value

	newSessionToken, err := ses.Update(sessionToken)
	webErrAndLog(w, err)


	// Set the new token as the users `session_token` cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   newSessionToken,
		Expires: time.Now().Add(120 * time.Second),
	})
}

func webErrAndLog(w http.ResponseWriter, err error) {
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
