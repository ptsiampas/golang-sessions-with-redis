package session

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
	"log"
)

var cache redis.Conn

type Session struct{
	DefaultSessionTimeout int
}

func init() {
	c, err := redis.DialURL("redis://localhost")
	if err != nil {
		log.Panicln(err)
	}

	cache = c
}

func (s Session) Create(v string) (string, error) {
	// Create a new random session token
	sessionToken, err := createToken()
	if err != nil {
		log.Fatalln(err)
	}

	// Set the token in the cache, along with the user whom it represents
	// The token has an expiry time of 120 seconds
	_, err = cache.Do("SETEX", sessionToken, s.DefaultSessionTimeout, v)
	if err != nil {
		return "", err
	}

	return sessionToken, nil
}

// Update Takes a response interface and the current token ID
func(s Session)Update(t string) (string, error) {
	// Get the existing token
	r, err := s.Get(t)
	if err != nil {
		log.Fatalln(err)
	}

	newSessionToken, err := createToken()
	if err != nil {
		log.Fatalln(err)
	}
	_, err = cache.Do("SETEX", newSessionToken, s.DefaultSessionTimeout, fmt.Sprintf("%s",r))
	if err != nil {
		return "", err
	}

	_, err = s.Delete(t)
	if err != nil {
		return "", err
	}
	return newSessionToken, nil
}

func (s Session) Get(t string) (interface{}, error) {
	r, err := cache.Do("GET", t)
	if err != nil {
		return nil, err
	}

	if r == nil {
		return nil, err
	}

	return r, err
}

func (s Session) Delete(t string) (interface{}, error) {
	n, err := cache.Do("DEL", t)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func validateSession() {
	return
}

func createToken() (string, error) {
	// Create a new random session token
	token, err := uuid.NewV4()
	if err != nil {
		log.Fatalln(err)
		return "", err
	}
	return token.String(), nil
}