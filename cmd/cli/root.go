package cli

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spiral/roadrunner/v2/plugins/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/spf13/cobra"
	"github.com/spiral/endure"
)

var (
	WorkDir   string
	CfgFile   string
	Container endure.Container
	Logger    *zap.Logger
	rootCmd   = &cobra.Command{
		Use:           "rr",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
)

// InitApp with a list of provided services.
func InitApp(service ...interface{}) error {
	var err error
	Container, err = endure.NewContainer(nil, endure.RetryOnFail(false), endure.SetLogLevel(endure.DebugLevel))
	if err != nil {
		return err
	}

	for _, svc := range service {
		err = Container.Register(svc)
		if err != nil {
			return err
		}
	}

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// exit with error, fatal invoke os.Exit(1)
		log.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&CfgFile, "config", "c", ".rr.yaml", "config file (default is .rr.yaml)")
	rootCmd.PersistentFlags().StringVarP(&WorkDir, "WorkDir", "w", "", "work directory")

	// todo: properly handle debug level
	Logger = initLogger()

	cobra.OnInitialize(func() {
		if CfgFile != "" {
			if absPath, err := filepath.Abs(CfgFile); err == nil {
				CfgFile = absPath

				// force working absPath related to config file
				if err := os.Chdir(filepath.Dir(absPath)); err != nil {
					panic(err)
				}
			}
		}

		if WorkDir != "" {
			if err := os.Chdir(WorkDir); err != nil {
				panic(err)
			}
		}

		// todo: config is global, not only for serve
		conf := &config.Viper{}
		conf.Path = CfgFile
		conf.Prefix = "rr"

		err := Container.Register(conf)
		if err != nil {
			panic(err)
		}
	})
}

func initLogger() *zap.Logger {
	// todo: we do not need it
	cfg := zap.Config{
		Level:    zap.NewAtomicLevelAt(zap.DebugLevel),
		Encoding: "console",
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:    "message",
			LevelKey:      "level",
			TimeKey:       "time",
			CallerKey:     "caller",
			NameKey:       "name",
			StacktraceKey: "stack",
			EncodeLevel:   zapcore.CapitalLevelEncoder,
			EncodeTime:    zapcore.ISO8601TimeEncoder,
			EncodeCaller:  zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := cfg.Build(zap.AddCaller())
	if err != nil {
		panic(err)
	}

	return logger
}
