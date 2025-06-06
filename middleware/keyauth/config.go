package keyauth

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

type KeyLookupFunc func(c fiber.Ctx) (string, error)

// Config defines the config for middleware.
type Config struct {
	// Next defines a function to skip middleware.
	// Optional. Default: nil
	Next func(fiber.Ctx) bool

	// SuccessHandler defines a function which is executed for a valid key.
	// Optional. Default: nil
	SuccessHandler fiber.Handler

	// ErrorHandler defines a function which is executed for an invalid key.
	// It may be used to define a custom error.
	// Optional. Default: 401 Invalid or expired key
	ErrorHandler fiber.ErrorHandler

	CustomKeyLookup KeyLookupFunc

	// Validator is a function to validate key.
	Validator func(fiber.Ctx, string) (bool, error)

	// KeyLookup is a string in the form of "<source>:<name>" that is used
	// to extract key from the request.
	// Optional. Default value "header:Authorization".
	// Possible values:
	// - "header:<name>"
	// - "query:<name>"
	// - "form:<name>"
	// - "param:<name>"
	// - "cookie:<name>"
	KeyLookup string

	// AuthScheme to be used in the Authorization header.
	// Optional. Default value "Bearer".
	AuthScheme string

	// Realm defines the protected area for WWW-Authenticate responses.
	// Optional. Default value "Restricted".
	Realm string
}

// ConfigDefault is the default config
var ConfigDefault = Config{
	SuccessHandler: func(c fiber.Ctx) error {
		return c.Next()
	},
	ErrorHandler:    nil,
	KeyLookup:       "header:" + fiber.HeaderAuthorization,
	CustomKeyLookup: nil,
	AuthScheme:      "Bearer",
	Realm:           "Restricted",
}

// Helper function to set default values
func configDefault(config ...Config) Config {
	// Return default config if nothing provided
	if len(config) < 1 {
		return ConfigDefault
	}

	// Override default config
	cfg := config[0]

	// Set default values
	if cfg.KeyLookup == "" {
		cfg.KeyLookup = ConfigDefault.KeyLookup
		if cfg.AuthScheme == "" {
			cfg.AuthScheme = ConfigDefault.AuthScheme
		}
	}
	if cfg.Realm == "" {
		cfg.Realm = ConfigDefault.Realm
	}
	if cfg.SuccessHandler == nil {
		cfg.SuccessHandler = ConfigDefault.SuccessHandler
	}
	if cfg.ErrorHandler == nil {
		localCfg := cfg
		cfg.ErrorHandler = func(c fiber.Ctx, err error) error {
			if localCfg.AuthScheme != "" {
				c.Set(fiber.HeaderWWWAuthenticate, fmt.Sprintf("%s realm=%q", localCfg.AuthScheme, localCfg.Realm))
			}
			if errors.Is(err, ErrMissingOrMalformedAPIKey) {
				return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
			}
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired API Key")
		}
	}
	if cfg.Validator == nil {
		panic("fiber: keyauth middleware requires a validator function")
	}

	return cfg
}
