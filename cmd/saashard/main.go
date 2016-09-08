package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"

	"github.com/berkaroad/saashard"
	"github.com/berkaroad/saashard/config"
	"github.com/berkaroad/saashard/server"
	"github.com/berkaroad/saashard/utils/golog"
)

var (
	configFile = flag.String("config", "/opt/saashard/conf/ss.yaml", "saashard config file")
	logLevel   = flag.String("log-level", "", "log level [debug|info|warn|error], default error")
	version    = flag.Bool("v", false, "the version of saashard")
)

const (
	sqlLogName = "sql.log"
	sysLogName = "sys.log"
	maxLogSize = 100 * 1024 * 1024
)

const banner string = `
                          __                   __
  __________  ____ ______/ /_  ____ __________/ /
  / ___// __\ / __\/ ___/ __ \ / __\/ ___/ __  /
 (__  ) /_/ / /_/ (__  ) / / / /_/ / /  / /_/ /
/____/\__,_/\__,_/____/_/ /_/\__,_/_/   \__,_/

`

func main() {
	fmt.Print(banner)
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()
	fmt.Printf("Git commit:%s\n", saashard.Version)
	fmt.Printf("Build time:%s\n", saashard.Compile)
	if *version {
		return
	}
	if len(*configFile) == 0 {
		fmt.Println("must use a config file")
		return
	}

	cfg, err := config.ParseConfigFile(*configFile)
	if err != nil {
		fmt.Printf("parse config file error:%v\n", err.Error())
		return
	}

	//when the log file size greater than 1GB, saashard will generate a new file
	if len(cfg.LogPath) != 0 {
		sysFilePath := path.Join(cfg.LogPath, sysLogName)
		sysFile, err := golog.NewRotatingFileHandler(sysFilePath, maxLogSize, 1)
		if err != nil {
			fmt.Printf("new log file error:%v\n", err.Error())
			return
		}
		golog.GlobalSysLogger = golog.New(sysFile, golog.Lfile|golog.Ltime|golog.Llevel)

		sqlFilePath := path.Join(cfg.LogPath, sqlLogName)
		sqlFile, err := golog.NewRotatingFileHandler(sqlFilePath, maxLogSize, 1)
		if err != nil {
			fmt.Printf("new log file error:%v\n", err.Error())
			return
		}
		golog.GlobalSqlLogger = golog.New(sqlFile, golog.Lfile|golog.Ltime|golog.Llevel)
	}

	if *logLevel != "" {
		setLogLevel(*logLevel)
	} else {
		setLogLevel(cfg.LogLevel)
	}

	var svr *server.Server
	svr, err = server.NewServer(cfg)
	if err != nil {
		golog.Error("main", "main", err.Error(), 0)
		golog.GlobalSysLogger.Close()
		golog.GlobalSqlLogger.Close()
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGPIPE,
	)

	go func() {
		for {
			sig := <-sc
			if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == syscall.SIGQUIT {
				golog.Info("main", "main", "Got signal", 0, "signal", sig)
				golog.GlobalSysLogger.Close()
				golog.GlobalSqlLogger.Close()
				svr.Close()
			} else if sig == syscall.SIGPIPE {
				golog.Info("main", "main", "Ignore broken pipe signal", 0)
			}
		}
	}()

	svr.Run()
}

func setLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		golog.GlobalSysLogger.SetLevel(golog.LevelDebug)
	case "info":
		golog.GlobalSysLogger.SetLevel(golog.LevelInfo)
	case "warn":
		golog.GlobalSysLogger.SetLevel(golog.LevelWarn)
	case "error":
		golog.GlobalSysLogger.SetLevel(golog.LevelError)
	default:
		golog.GlobalSysLogger.SetLevel(golog.LevelError)
	}
}