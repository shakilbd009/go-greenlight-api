package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/shakilbd009/go-greenlight-api/internal/data"
	"github.com/shakilbd009/go-greenlight-api/internal/jsonlog"
	"github.com/shakilbd009/go-greenlight-api/internal/mailer"
)

// Create a buildTime variable to hold the executable binary build time. Note that this
// must be a string type, as the -X linker flag will only work with string variables.
var (
	buildTime string
	version   string
)

// Define a config struct to hold all the configuration settings for our application.
// For now, the only configuration settings will be the network port that we want the
// server to listen on, and the name of the current operating environment for the
// application (development, staging, production, etc.). We will read in these
// configuration settings from command-line flags when the application starts.
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
	}
	// Add a new limiter struct containing fields for the requests-per-second and burst
	// values, and a boolean field which we can use to enable/disable rate limiting
	// altogether.
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}

	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

// Define an application struct to hold the dependencies for our HTTP handlers, helpers,
// and middleware. At the moment this only contains a copy of the config struct and a
// logger, but it will grow to include a lot more as our build progresses.
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	// Declare an instance of the config struct.
	var cfg config
	var ori string
	// Read the value of the port and env command-line flags into the config struct. We
	// default to using the port number 4000 and the environment "development" if no
	// corresponding flags are provided.
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	// env := os.Getenv("GREENLIGHT_DB_DSN")
	// if env == "" {
	// 	logger.PrintFatal(fmt.Errorf("%s env variable is not set", "GREENLIGHT_DB_DSN"), nil)
	// }
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "Postgres DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	// Create command line flags to read the setting values into the config struct.
	// Notice that we use true as the default for the 'enabled' setting?
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	// Read the SMTP server configuration settings into the config struct, using the
	// Mailtrap settings as the default values. IMPORTANT: If you're following along,
	// make sure to replace the default values for smtp-username and smtp-password
	// with your own Mailtrap credentials.
	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "d07b83269b2b20", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "cf5f2176a12f39", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")
	flag.StringVar(&ori, "cors-trusted-origins", "", "Trusted CORS origins (space separated)")

	// Create a new version boolean flag with the default value of false.
	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()
	// If the version flag value is true, then print out the version number and // immediately exit.
	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		// Print out the contents of the buildTime variable.
		fmt.Printf("Build time:\t%s\n", buildTime)
		os.Exit(0)
	}
	cfg.cors.trustedOrigins = strings.Fields(ori)

	// Initialize a new jsonlog.Logger which writes any messages *at or above* the INFO
	// severity level to the standard out stream.

	db, err := openDB(cfg)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	logger.PrintInfo("database connection pool established", nil)
	// Publish a new "version" variable in the expvar handler containing our application // version number (currently the constant "1.0.0").
	expvar.NewString("version").Set(version)
	expvar.Publish("go-routine", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))
	// Publish the database connection pool statistics.
	expvar.Publish("database", expvar.Func(func() interface{} {
		return db.Stats()
	}))
	// Publish the current Unix timestamp.
	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))
	// Declare an instance of the application struct, containing the config struct and
	// the logger.
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}

}

// The openDB() function returns a sql.DB connection pool.
func openDB(cfg config) (*sql.DB, error) {
	// Use sql.Open() to create an empty connection pool, using the DSN from the config // struct.
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Use PingContext() to establish a new connection to the database, passing in the
	// context we created above as a parameter. If the connection couldn't be
	// established successfully within the 5 second deadline, then this will return an // error.
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	// Return the sql.DB connection pool.
	return db, nil
}
