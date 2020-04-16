package main

import (
	"context"
	"crypto/rsa"
	"expvar"
	"fmt"
	"github.com/ardanlabs/conf"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/remisb/restaurant/cmd/restaurant-api/internal/handlers"
	"github.com/remisb/restaurant/internal/platform/auth"
	"github.com/remisb/restaurant/internal/platform/database"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// build is the git version of this program. It is set using builcd flags in the makefile.
var build = "develop"

func main() {
	if err := run(); err != nil {
		log.Println("error :", err)
		os.Exit(1)
	}
}

func run() error {
	log := log.New(os.Stdout, "SALES : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	var cfg struct {
		Web struct {
			APIHost         string
			DebugHost       string
			ReadTimeout     time.Duration
			WriteTimeout    time.Duration
			ShutdownTimeout time.Duration
		}
		DB struct {
			User       string `conf:"default:postgres"`
			Password   string `conf:"default:postgres,noprint"`
			Host       string `conf:"default:0.0.0.0"`
			Name       string `conf:"default:postgres"`
			DisableTLS bool   `conf:"default:false"`
		}
		Auth struct {
			KeyID          string `conf:"default:1"`
			PrivateKeyFile string `conf:"default:/app/private.pem"`
			Algorithm      string `conf:"default:RS256"`
		}
	}

	if err := conf.Parse(os.Args[1:], "RESTAURANT", &cfg); err != nil {
		if err == conf.ErrHelpWanted {
			usage, err := conf.Usage("RESTAURANT", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			return nil
		}
		return errors.Wrap(err, "parsing config")
	}

	// App Starting
	expvar.NewString("build").Set(build)
	log.Printf("main : Started : Application initializing : version %q", build)
	defer log.Println("main : Completed")

	out, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}
	log.Printf("main : Config :\n%v\n", out)

	// Initialize authentication support

	log.Println("main : Started : Initializing authentication support")
	keyContents, err := ioutil.ReadFile(cfg.Auth.PrivateKeyFile)
	if err != nil {
		return errors.Wrap(err, "reading auth private key")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyContents)
	if err != nil {
		return errors.Wrap(err, "constructing authenticator")
	}

	f := auth.NewSimpleKeyLookupFunc(cfg.Auth.KeyID, privateKey.Public().(*rsa.PublicKey))
	authenticator, err := auth.NewAuthenticator(privateKey, cfg.Auth.KeyID, cfg.Auth.Algorithm, f)
	if err != nil {
		return errors.Wrap(err, "constructing authenticator")
	}

	// Start Database

	log.Println("main . Started : Initializing database support")

	db, err := database.Open(database.Config{
		User:       cfg.DB.User,
		Password:   cfg.DB.Password,
		Host:       cfg.DB.Host,
		Name:       cfg.DB.Name,
		DisableTLS: cfg.DB.DisableTLS,
	})
	if err != nil {
		return errors.Wrap(err, "connecting to db")
	}
	defer func() {
		log.Printf("main : Database Stopping : %s", cfg.DB.Host)
	}()

	// Start Tracing Support

	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.
	//
	// Not concerned with shutting this down when the application is shutdown.

	log.Println("main : Started : Initializing debugging support" )

	go func() {
		log.Printf("main : Debug Listening %s", cfg.Web.DebugHost)
		log.Printf("main : Debug Listener closed : %v", http.ListenAndServe(cfg.Web.DebugHost, http.DefaultServeMux))
	}()

	// Start API Service

	log.Println("main : Started : Initializing API support")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	api := http.Server{
		Addr: cfg.Web.APIHost,
		Handler: handlers.API(build, shutdown, log, db, authenticator),
		ReadTimeout: cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("main : API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// Shutdown

	select {
	case err := <- serverErrors:
		return errors.Wrap(err, "server error")

		case sig := <- shutdown:
			log.Printf("main : %v : Start shtdown", sig)

			ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
			defer cancel()

			err := api.Shutdown(ctx)
			if err != nil {
				log.Printf("main : Graceful shutdown did not complete in %v : %v", cfg.Web.ShutdownTimeout, err)
				err = api.Close()
			}

			switch {
			case sig == syscall.SIGSTOP:
				return errors.New("integrity issue caused shutdown")
			case err != nil:
				return errors.Wrap(err, "could not stop server gracefully")
			}
	}
	return nil
}
