package session
// Tests with encoding on play https://play.golang.org/p/EmYoP2p78F5
import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
	"log"
	"strings"
)

var cache redis.Conn

type Session struct {
	DefaultSessionTimeout int
	HmacKey               string
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
	sessionToken, err := s.createToken()
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
func (s Session) Update(t string) (string, error) {
	// Get the existing token
	r, err := s.Get(t)
	if err != nil {
		log.Fatalln(err)
	}

	newSessionToken, err := s.createToken()
	if err != nil {
		log.Fatalln(err)
	}
	_, err = cache.Do("SETEX", newSessionToken, s.DefaultSessionTimeout, fmt.Sprintf("%s", r))
	if err != nil {
		return "", err
	}

	_, err = s.delete(t)
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

	return s.decodeSessionValue(t), err
}

func (s Session) delete(t string) (interface{}, error) {
	n, err := cache.Do("DEL", t)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s Session) createToken() (string, error) {
	// Create a new random session token
	token, err := uuid.NewV4()
	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	return s.encodeSessionValue(token.String()), nil
}

func (s Session) encodeSessionValue(esv string) string {
	h := hmac.New(sha256.New, []byte(s.HmacKey))
	h.Write([]byte(esv))

	hmacValue := hex.EncodeToString(h.Sum(nil))
	//fmt.Println("hmacValue", hmacValue)

	encodedValue := esv + "|" + hmacValue
	fmt.Println("String to Encode", encodedValue)
	s64 := base64.StdEncoding.EncodeToString([]byte(encodedValue))

	return s64
}

func (s Session) decodeSessionValue(dsv string) string {

	bs, err := base64.StdEncoding.DecodeString(dsv)
	if err != nil {
		log.Fatalln("Something went wrong with the decoding", err)
	}
	items := strings.Split(string(bs), "|")
	if len(items) != 2 {
		log.Println("invalid Session String")
		return ""
	}

	message := []byte(items[0])
	messageMAC, _ := hex.DecodeString(items[1])

	b := validMAC(message, messageMAC, []byte(s.HmacKey))
	if b != true {
		fmt.Println("Invalid Mac")
		return ""
	}
	fmt.Println("Items", items)

	return string(message)
}

func validMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMAC, expectedMAC)
}
