package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/ardanlabs/kit/cfg"
	"github.com/ardanlabs/kit/log"
	"github.com/boltdb/bolt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"
)

const errMsgBucket = "error_messages"

var errInvalidToken = errors.New("token invalid")

// CLISetup sets up the logging and loads the configuration.
func CLISetup(c *cli.Context) {
	// if the configuration file is not specified
	if c.GlobalString("config") == "" {
		// then load the config from the environment
		cfg.Init(cfg.EnvProvider{Namespace: "ERRCATCH"})
	} else {
		// then load the config from a file
		cfg.Init(cfg.FileProvider{Filename: c.GlobalString("config")})
	}

	log.Init(os.Stdout, func() int {
		env, _ := cfg.String("ENV")

		if env == "production" {
			return log.USER
		}

		return log.DEV
	}, log.Ldate|log.Ltime|log.Lmicroseconds)
}

// CLIServe serves the webserver for errcatch.
func CLIServe(c *cli.Context) error {
	CLISetup(c)

	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err := bolt.Open(cfg.MustString("DB"), 0600, nil)
	if err != nil {
		log.Fatal("main", "CLIServe", "Can't open database: %s", err.Error())
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(errMsgBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		log.Fatal("main", "CLIServe", "Can't update database: %s", err.Error())
	}

	a := App{db: db}

	env, _ := cfg.String("ENV")

	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	mux := gin.Default()

	listHTML, err := Asset("templates/list.html")
	if err != nil {
		log.Fatal("main", "CLIServe", "Can't load template: %s", err.Error())
	}

	listTemplate := template.Must(template.New("list").Parse(string(listHTML)))

	mux.SetHTMLTemplate(listTemplate)

	auth := gin.BasicAuth(gin.Accounts{
		cfg.MustString("USER"): cfg.MustString("PASSWORD"),
	})

	mux.GET("/", auth, a.ListErrors)
	mux.POST("/error", a.AddError)
	mux.DELETE("/error/:id", auth, a.RemoveError)

	bind := cfg.MustString("BIND")

	log.User("main", "CLIServe", "Now serving on %s", bind)
	http.ListenAndServe(bind, mux)

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "errcatch"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Usage: "specify a config `FILE` to use",
		},
	}
	app.Commands = []cli.Command{
		{
			Name: "sign",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "secret",
				},
				cli.StringFlag{
					Name: "app",
				},
			},
			Action: func(c *cli.Context) error {
				claims := AppClaims{
					App: c.String("app"),
				}

				token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

				ss, err := token.SignedString([]byte(c.String("secret")))
				if err != nil {
					// log.Println(err)
					return err
				}

				fmt.Println(ss)

				return nil
			},
		},
		{
			Name:   "serve",
			Action: CLIServe,
		},
	}

	app.Run(os.Args)
}
