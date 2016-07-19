package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/ardanlabs/kit/cfg"
	"github.com/boltdb/bolt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

// AppClaims are the claims about an App trying to report an Error.
type AppClaims struct {
	App string
	jwt.StandardClaims
}

// App contains the database handle.
type App struct {
	db *bolt.DB
}

func (a *App) Error(c *gin.Context, err error) {
	if err == errInvalidToken {
		c.Status(http.StatusUnauthorized)
	} else {
		c.Status(http.StatusInternalServerError)
	}

	log.Printf("ERROR: %#v\n", err)
}

// ListErrors lists the errors in the database.
func (a *App) ListErrors(c *gin.Context) {
	var errorMessages = make([]ErrorMsg, 0)

	err := a.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(errMsgBucket))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var errMsg ErrorMsg
			err := json.Unmarshal(v, &errMsg)
			if err != nil {
				return err
			}

			errorMessages = append(errorMessages, errMsg)
		}

		return nil
	})
	if err != nil {
		a.Error(c, err)
		return
	}

	for i, errMsg := range errorMessages {
		b, err := json.MarshalIndent(errMsg.Payload, "", "  ")
		if err != nil {
			a.Error(c, err)
			return
		}

		errorMessages[i].PayloadJSON = string(b)
	}

	// sort the errors by their created at time stamp
	sort.Sort(ByCreatedAt(errorMessages))

	c.HTML(http.StatusOK, "list", errorMessages)
}

// RemoveError removes the error from the database.
func (a *App) RemoveError(c *gin.Context) {
	err := a.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(errMsgBucket))
		return b.Delete([]byte(c.Param("id")))
	})
	if err != nil {
		a.Error(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// AddError adds a new error to the database.
func (a *App) AddError(c *gin.Context) {
	token, err := jwt.ParseWithClaims(c.Request.Header.Get("Authorization"), &AppClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.MustString("SECRET")), nil
	})

	claims, ok := token.Claims.(*AppClaims)
	if !ok || !token.Valid {
		a.Error(c, err)
		return
	}

	var payload map[string]interface{}

	err = c.BindJSON(&payload)
	if err != nil {
		a.Error(c, err)
		return
	}

	var errMsg = ErrorMsg{
		ID:        uuid.NewV4().String(),
		App:       claims.App,
		CreatedAt: time.Now(),
		Payload:   payload,
	}

	buf, err := json.Marshal(errMsg)
	if err != nil {
		a.Error(c, err)
		return
	}

	err = a.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(errMsgBucket))
		return b.Put([]byte(errMsg.ID), buf)
	})
	if err != nil {
		a.Error(c, err)
		return
	}

	var response = map[string]string{
		"code": errMsg.ID,
	}

	c.JSON(http.StatusCreated, response)
}
