package rewrite

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func Test_New(t *testing.T) {
	// Test with no config
	m := New()

	if m == nil {
		t.Error("Expected middleware to be returned, got nil")
	}

	// Test with config
	m = New(Config{
		Rules: map[string]string{
			"/old": "/new",
		},
	})

	if m == nil {
		t.Error("Expected middleware to be returned, got nil")
	}

	// Test with full config
	m = New(Config{
		Next: func(fiber.Ctx) bool {
			return true
		},
		Rules: map[string]string{
			"/old": "/new",
		},
	})

	if m == nil {
		t.Error("Expected middleware to be returned, got nil")
	}
}

func Test_Rewrite(t *testing.T) {
	// Case 1: Next function always returns true
	app := fiber.New()
	app.Use(New(Config{
		Next: func(fiber.Ctx) bool {
			return true
		},
		Rules: map[string]string{
			"/old": "/new",
		},
	}))

	app.Get("/old", func(c fiber.Ctx) error {
		return c.SendString("Rewrite Successful")
	})

	req, err := http.NewRequestWithContext(context.Background(), fiber.MethodGet, "/old", nil)
	require.NoError(t, err)
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyString := string(body)

	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Equal(t, "Rewrite Successful", bodyString)

	// Case 2: Next function always returns false
	app = fiber.New()
	app.Use(New(Config{
		Next: func(fiber.Ctx) bool {
			return false
		},
		Rules: map[string]string{
			"/old": "/new",
		},
	}))

	app.Get("/new", func(c fiber.Ctx) error {
		return c.SendString("Rewrite Successful")
	})

	req, err = http.NewRequestWithContext(context.Background(), fiber.MethodGet, "/old", nil)
	require.NoError(t, err)
	resp, err = app.Test(req)
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyString = string(body)

	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Equal(t, "Rewrite Successful", bodyString)

	// Case 3: check for captured tokens in rewrite rule
	app = fiber.New()
	app.Use(New(Config{
		Rules: map[string]string{
			"/users/*/orders/*": "/user/$1/order/$2",
		},
	}))

	app.Get("/user/:userID/order/:orderID", func(c fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("User ID: %s, Order ID: %s", c.Params("userID"), c.Params("orderID")))
	})

	req, err = http.NewRequestWithContext(context.Background(), fiber.MethodGet, "/users/123/orders/456", nil)
	require.NoError(t, err)
	resp, err = app.Test(req)
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyString = string(body)

	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Equal(t, "User ID: 123, Order ID: 456", bodyString)

	// Case 4: Send non-matching request, handled by default route
	app = fiber.New()
	app.Use(New(Config{
		Rules: map[string]string{
			"/users/*/orders/*": "/user/$1/order/$2",
		},
	}))

	app.Get("/user/:userID/order/:orderID", func(c fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("User ID: %s, Order ID: %s", c.Params("userID"), c.Params("orderID")))
	})

	app.Use(func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req, err = http.NewRequestWithContext(context.Background(), fiber.MethodGet, "/not-matching-any-rule", nil)
	require.NoError(t, err)
	resp, err = app.Test(req)
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyString = string(body)

	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Equal(t, "OK", bodyString)

	// Case 4: Send non-matching request, with no default route
	app = fiber.New()
	app.Use(New(Config{
		Rules: map[string]string{
			"/users/*/orders/*": "/user/$1/order/$2",
		},
	}))

	app.Get("/user/:userID/order/:orderID", func(c fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("User ID: %s, Order ID: %s", c.Params("userID"), c.Params("orderID")))
	})

	req, err = http.NewRequestWithContext(context.Background(), fiber.MethodGet, "/not-matching-any-rule", nil)
	require.NoError(t, err)
	resp, err = app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func Benchmark_Rewrite(b *testing.B) {
	// Helper function to create a new Fiber app with rewrite middleware
	createApp := func(config Config) *fiber.App {
		app := fiber.New()
		app.Use(New(config))
		return app
	}

	// Benchmark: Rewrite with Next function always returns true
	b.Run("Next always true", func(b *testing.B) {
		app := createApp(Config{
			Next: func(fiber.Ctx) bool {
				return true
			},
			Rules: map[string]string{
				"/old": "/new",
			},
		})

		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/old")
		b.ReportAllocs()
		for b.Loop() {
			app.Handler()(reqCtx)
		}
	})

	// Benchmark: Rewrite with Next function always returns false
	b.Run("Next always false", func(b *testing.B) {
		app := createApp(Config{
			Next: func(fiber.Ctx) bool {
				return false
			},
			Rules: map[string]string{
				"/old": "/new",
			},
		})

		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/old")
		b.ReportAllocs()
		for b.Loop() {
			app.Handler()(reqCtx)
		}
	})

	// Benchmark: Rewrite with tokens
	b.Run("Rewrite with tokens", func(b *testing.B) {
		app := createApp(Config{
			Rules: map[string]string{
				"/users/*/orders/*": "/user/$1/order/$2",
			},
		})

		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/users/123/orders/456")
		b.ReportAllocs()
		for b.Loop() {
			app.Handler()(reqCtx)
		}
	})

	// Benchmark: Non-matching request, handled by default route
	b.Run("NonMatch with default", func(b *testing.B) {
		app := createApp(Config{
			Rules: map[string]string{
				"/users/*/orders/*": "/user/$1/order/$2",
			},
		})
		app.Use(func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/not-matching-any-rule")
		b.ReportAllocs()
		for b.Loop() {
			app.Handler()(reqCtx)
		}
	})

	// Benchmark: Non-matching request, with no default route
	b.Run("NonMatch without default", func(b *testing.B) {
		app := createApp(Config{
			Rules: map[string]string{
				"/users/*/orders/*": "/user/$1/order/$2",
			},
		})

		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/not-matching-any-rule")
		b.ReportAllocs()
		for b.Loop() {
			app.Handler()(reqCtx)
		}
	})
}

func Benchmark_Rewrite_Parallel(b *testing.B) {
	// Helper function to create a new Fiber app with rewrite middleware
	createApp := func(config Config) *fiber.App {
		app := fiber.New()
		app.Use(New(config))
		return app
	}

	// Parallel Benchmark: Rewrite with Next function always returns true
	b.Run("Next always true", func(b *testing.B) {
		app := createApp(Config{
			Next: func(fiber.Ctx) bool {
				return true
			},
			Rules: map[string]string{
				"/old": "/new",
			},
		})
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.SetRequestURI("/old")
			for pb.Next() {
				app.Handler()(reqCtx)
			}
		})
	})

	// Parallel Benchmark: Rewrite with Next function always returns false
	b.Run("Next always false", func(b *testing.B) {
		app := createApp(Config{
			Next: func(fiber.Ctx) bool {
				return false
			},
			Rules: map[string]string{
				"/old": "/new",
			},
		})
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.SetRequestURI("/old")
			for pb.Next() {
				app.Handler()(reqCtx)
			}
		})
	})

	// Parallel Benchmark: Rewrite with tokens
	b.Run("Rewrite with tokens", func(b *testing.B) {
		app := createApp(Config{
			Rules: map[string]string{
				"/users/*/orders/*": "/user/$1/order/$2",
			},
		})
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.SetRequestURI("/users/123/orders/456")
			for pb.Next() {
				app.Handler()(reqCtx)
			}
		})
	})

	// Parallel Benchmark: Non-matching request, handled by default route
	b.Run("NonMatch with default", func(b *testing.B) {
		app := createApp(Config{
			Rules: map[string]string{
				"/users/*/orders/*": "/user/$1/order/$2",
			},
		})
		app.Use(func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.SetRequestURI("/not-matching-any-rule")
			for pb.Next() {
				app.Handler()(reqCtx)
			}
		})
	})

	// Parallel Benchmark: Non-matching request, with no default route
	b.Run("NonMatch without default", func(b *testing.B) {
		app := createApp(Config{
			Rules: map[string]string{
				"/users/*/orders/*": "/user/$1/order/$2",
			},
		})
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.SetRequestURI("/not-matching-any-rule")
			for pb.Next() {
				app.Handler()(reqCtx)
			}
		})
	})
}
