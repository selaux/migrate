package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/mattes/migrate"
)

// set main log
var log = &Log{}

func main() {
	helpPtr := flag.Bool("help", false, "")
	versionPtr := flag.Bool("version", false, "")
	verbosePtr := flag.Bool("verbose", false, "")
	prefetchPtr := flag.Uint("prefetch", 10, "")
	lockTimeoutPtr := flag.Uint("lock-timeout", 15, "")
	pathPtr := flag.String("path", "", "")
	databasePtr := flag.String("database", "", "")
	sourcePtr := flag.String("source", "", "")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr,
			`Usage: migrate OPTIONS COMMAND [arg...]
       migrate [ -version | -help ]

Options:
  -source          Location of the migrations (driver://url)
  -path            Shorthand for -source=file://path 
  -database        Run migrations against this database (driver://url)
  -prefetch N      Number of migrations to load in advance before executing (default 10)
  -lock-timeout N  Allow N seconds to acquire database lock (default 15)
  -verbose         Print verbose logging
  -version         Print version
  -help            Print usage

Commands:
  goto V       Migrate to version V
  up [N]       Apply all or N up migrations
  down [N]     Apply all or N down migrations
  drop         Drop everyting inside database
  force V      Set version V but don't run migration (ignores dirty state)
  version      Print current migration version
`)
	}

	flag.Parse()

	// initialize logger
	log.verbose = *verbosePtr

	// show cli version
	if *versionPtr {
		fmt.Fprintln(os.Stderr, Version)
		os.Exit(0)
	}

	// show help
	if *helpPtr {
		flag.Usage()
		os.Exit(0)
	}

	// translate -path into -source if given
	if *sourcePtr == "" && *pathPtr != "" {
		*sourcePtr = fmt.Sprintf("file://%v", filepath.Clean(*pathPtr))
	}

	// initialize migrate
	// don't catch migraterErr here and let each command decide
	// how it wants to handle the error
	migrater, migraterErr := migrate.New(*sourcePtr, *databasePtr)
	defer func() {
		if migraterErr == nil {
			migrater.Close()
		}
	}()
	if migraterErr == nil {
		migrater.Log = log
		migrater.PrefetchMigrations = *prefetchPtr
		migrater.LockTimeout = time.Duration(int64(*lockTimeoutPtr)) * time.Second

		// handle Ctrl+c
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT)
		go func() {
			for range signals {
				log.Println("Stopping after this running migration ...")
				migrater.GracefulStop <- true
				return
			}
		}()
	}

	startTime := time.Now()

	switch flag.Arg(0) {
	case "goto":
		if migraterErr != nil {
			log.fatalErr(migraterErr)
		}

		if flag.Arg(1) == "" {
			log.fatal("error: please specify version argument V")
		}

		v, err := strconv.ParseUint(flag.Arg(1), 10, 64)
		if err != nil {
			log.fatal("error: can't read version argument V")
		}

		gotoCmd(migrater, uint(v))

		if log.verbose {
			log.Println("Finished after", time.Now().Sub(startTime))
		}

	case "up":
		if migraterErr != nil {
			log.fatalErr(migraterErr)
		}

		limit := -1
		if flag.Arg(1) != "" {
			n, err := strconv.ParseUint(flag.Arg(1), 10, 64)
			if err != nil {
				log.fatal("error: can't read limit argument N")
			}
			limit = int(n)
		}

		upCmd(migrater, limit)

		if log.verbose {
			log.Println("Finished after", time.Now().Sub(startTime))
		}

	case "down":
		if migraterErr != nil {
			log.fatalErr(migraterErr)
		}

		limit := -1
		if flag.Arg(1) != "" {
			n, err := strconv.ParseUint(flag.Arg(1), 10, 64)
			if err != nil {
				log.fatal("error: can't read limit argument N")
			}
			limit = int(n)
		}

		downCmd(migrater, limit)

		if log.verbose {
			log.Println("Finished after", time.Now().Sub(startTime))
		}

	case "drop":
		if migraterErr != nil {
			log.fatalErr(migraterErr)
		}

		dropCmd(migrater)

		if log.verbose {
			log.Println("Finished after", time.Now().Sub(startTime))
		}

	case "force":
		if migraterErr != nil {
			log.fatalErr(migraterErr)
		}

		if flag.Arg(1) == "" {
			log.fatal("error: please specify version argument V")
		}

		v, err := strconv.ParseUint(flag.Arg(1), 10, 64)
		if err != nil {
			log.fatal("error: can't read version argument V")
		}

		forceCmd(migrater, uint(v))

		if log.verbose {
			log.Println("Finished after", time.Now().Sub(startTime))
		}

	case "version":
		if migraterErr != nil {
			log.fatalErr(migraterErr)
		}

		versionCmd(migrater)

	default:
		flag.Usage()
		os.Exit(0)
	}
}