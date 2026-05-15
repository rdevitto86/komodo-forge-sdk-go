package moxtox

import (
	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	dir        string
	config     MoxtoxConfig
	once       sync.Once
	allowMocks = true
)

const loggerPrefix = "[::Moxtox::]"

// Intercepts real HTTP requests and replaces them with configured mock responses.
func InitMoxtoxMiddleware(env string, configPath ...string) func(http.Handler) http.Handler {
	once.Do(func() {
		// Determine baseDir from configPath or default
		if len(configPath) == 0 || configPath[0] == "" {
			cwd, err := os.Getwd()
			if err != nil {
				logger.Error(loggerPrefix+" unable to determine current working directory", err)
				allowMocks = false
				return
			}
			dir = filepath.Join(cwd, "test", "moxtox")
		} else {
			dir = configPath[0]
		}

		// Load the Moxtox config
		if data, err := os.ReadFile(filepath.Join(dir, "moxtox_config.yml")); err == nil {
			if err := yaml.Unmarshal(data, &config); err != nil {
				logger.Error(loggerPrefix+"error loading moxtox config", err)
				allowMocks = false
				return
			}
			if !config.EnableMoxtox {
				logger.Info(loggerPrefix + " mocks disabled - using default behavior")
				allowMocks = false
				return
			}
			if !slices.Contains(config.AllowedEnvironments, env) {
				logger.Info(loggerPrefix + " mocks not allowed in this environment")
				allowMocks = false
				return
			}

			// build mock data store based on mode
			switch config.PerformanceMode {
			case "quick":
				config.buildHashLookupMap()
			case "dynamic":
				totalScenarios := config.countTotalScenarios()
				if totalScenarios > 10 { // threshold for switching to quick mode
					config.buildHashLookupMap()
				} else {
					config.buildSliceLookupMap()
				}
			default: // "default"
				config.buildSliceLookupMap()
			}

			logger.Info(loggerPrefix + " mocks enabled successfully")
		} else {
			logger.Error(loggerPrefix+" error loading moxtox config", err)
			allowMocks = false
		}
	})

	// ignore if mocks are disabled
	if allowMocks {
		return mockResponseHandler()
	}
	return func(next http.Handler) http.Handler { return next }
}

func mockResponseHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
			// Check for ignored routes first
			if slices.Contains(config.IgnoredRoutes, req.URL.Path) {
				next.ServeHTTP(wtr, req)
				return
			}

			// Use LookupMap for lookup
			if scenario, ok := matchesRequest(req); ok {
				if err := injectMock(wtr, req, scenario); err != nil {
					httpErr.SendError(
						wtr, req, httpErr.Global.UnprocessableEntity,
						httpErr.WithDetail("failed to inject mock"),
					)
					return
				}
				return
			}

			// No match: return 418 Teapot error
			httpErr.SendCustomError(wtr, req, http.StatusTeapot, "no mocks found", "", "MOXTOX_001")
		})
	}
}
