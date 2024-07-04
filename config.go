package apitool

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/wayne011872/api-toolkit/auth"
	"github.com/wayne011872/api-toolkit/errors"
	"github.com/wayne011872/api-toolkit/mid"
)

// Retrieve config from environmental variables

// Configuration will be pulled from the environment using the following keys
const (
	envApiPort = "API_PORT"

	envGinMode    = "GIN_MODE"
	envService    = "SERVICE"
	envIsMockAuth = "MOCK_AUTH"
	envIsDebug    = "API_DEBUG"

	envTrustedProxies = "TRUSTED_PROXIES"
)

// config holds the configuration
type Config struct {
	Service        string
	GinMode        string
	IsMockAuth     bool
	ApiPort        int
	TrustedProxies []string
	Debug          bool // autopaho and paho debug output requested

	proms          []prometheus.Collector
	authMid        auth.GinAuthMidInter
	preAuthMiddles []mid.GinMiddle
	middles        []mid.GinMiddle
	apis           []GinAPI
	errorHandler   errors.GinServerErrorHandler
	Logger         Log
}

func (cfg *Config) SetServerErrorHandler(handler errors.GinServerErrorHandler) {
	cfg.errorHandler = handler
}

func (cfg *Config) SetAuth(authmid auth.GinAuthMidInter) {
	cfg.authMid = authmid
}

func (cfg *Config) SetPreAuthMiddles(mids ...mid.GinMiddle) {
	cfg.preAuthMiddles = mids
}

func (cfg *Config) SetMiddles(mids ...mid.GinMiddle) {
	cfg.middles = mids
}

func (cfg *Config) SetAPIs(apis ...GinAPI) {
	cfg.apis = apis
}

func (cfg *Config) getMiddles() []mid.GinMiddle {
	count := 0
	var middles []mid.GinMiddle
	hasAuth := cfg.authMid != nil
	if cfg.Debug && hasAuth {
		middles = make([]mid.GinMiddle, len(cfg.preAuthMiddles)+len(cfg.middles)+2)
		middles[0] = mid.NewGinDebugMid()
		count = 1
	} else if hasAuth {
		middles = make([]mid.GinMiddle, len(cfg.preAuthMiddles)+len(cfg.middles)+1)
	} else {
		middles = make([]mid.GinMiddle, len(cfg.preAuthMiddles)+len(cfg.middles))
	}
	for i := 0; i < len(cfg.preAuthMiddles); i++ {
		middles[count] = cfg.preAuthMiddles[i]
		count++
	}
	if hasAuth {
		middles[count] = cfg.authMid
		count++
	}

	for i := 0; i < len(cfg.middles); i++ {
		middles[count] = cfg.middles[i]
		count++
	}
	return middles
}

func (cfg *Config) AddProms(c ...prometheus.Collector) {
	cfg.proms = c
}

// getConfig - Retrieves the configuration from the environment
func GetConfigFromEnv() (*Config, error) {
	var cfg Config
	var err error

	name, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	cfg.Service, err = stringFromEnv(envService)
	if err != nil {
		return nil, err
	}
	cfg.Service = fmt.Sprintf("%s-%s", cfg.Service, name)

	cfg.GinMode, err = stringFromEnv(envGinMode)
	if err != nil {
		return nil, err
	}

	cfg.ApiPort, err = intFromEnv(envApiPort)
	if err != nil {
		return nil, err
	}

	cfg.IsMockAuth, err = booleanFromEnv(envIsMockAuth)
	if err != nil {
		return nil, err
	}

	if cfg.IsMockAuth {
		cfg.SetAuth(auth.NewMockAuthMid())
	}

	cfg.Debug, err = booleanFromEnv(envIsDebug)
	if err != nil {
		return nil, err
	}

	proxies, err := stringFromEnv(envTrustedProxies)
	if err != nil {
		return nil, err
	}
	cfg.TrustedProxies = strings.Split(proxies, ",")

	return &cfg, nil
}

// stringFromEnv - Retrieves a string from the environment and ensures it is not blank (ort non-existent)
func stringFromEnv(key string) (string, error) {
	s := os.Getenv(key)
	if len(s) == 0 {
		return "", fmt.Errorf("environmental variable %s must not be blank", key)
	}
	return s, nil
}

// intFromEnv - Retrieves an integer from the environment (must be present and valid)
func intFromEnv(key string) (int, error) {
	s := os.Getenv(key)
	if len(s) == 0 {
		return 0, fmt.Errorf("environmental variable %s must not be blank", key)
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("environmental variable %s must be an integer", key)
	}
	return i, nil
}

// milliSecondsFromEnv - Retrieves milliseconds (as time.Duration) from the environment (must be present and valid)
// func milliSecondsFromEnv(key string) (time.Duration, error) {
// 	s := os.Getenv(key)
// 	if len(s) == 0 {
// 		return 0, fmt.Errorf("environmental variable %s must not be blank", key)
// 	}
// 	i, err := strconv.Atoi(s)
// 	if err != nil {
// 		return 0, fmt.Errorf("environmental variable %s must be an integer", key)
// 	}
// 	return time.Duration(i) * time.Millisecond, nil
// }

// booleanFromEnv - Retrieves boolean from the environment (must be present and valid)
func booleanFromEnv(key string) (bool, error) {
	s := os.Getenv(key)
	if len(s) == 0 {
		return false, fmt.Errorf("environmental variable %s must not be blank", key)
	}
	switch strings.ToUpper(s) {
	case "TRUE", "T", "1":
		return true, nil
	case "FALSE", "F", "0":
		return false, nil
	default:
		return false, fmt.Errorf("environmental variable %s be a valid boolean option (is %s)", key, s)
	}
}

type Log interface {
	Infof(format string, a ...any)
	Fatalf(format string, a ...any)
}
