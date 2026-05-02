// Package main es el composition root del API.
//
// Responsabilidad única: instanciar y conectar los componentes de las capas
// internas + orquestar el ciclo de vida del proceso (start, signals, shutdown).
// NO contiene lógica de negocio.
//
// Patrón de DI: Manual Dependency Injection. Sin frameworks (Wire, Fx). Para
// 4 use cases + 1 repo + 1 router, hacerlo a mano son ~10 líneas y se lee
// mejor que cualquier generación de código mágica.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lmarinaro/marinaro-hackerrank/internal/application"
	httpadapter "github.com/lmarinaro/marinaro-hackerrank/internal/infrastructure/http"
	"github.com/lmarinaro/marinaro-hackerrank/internal/infrastructure/persistence"
)

// Timeouts del HTTP server.
//
// Por qué fijarlos explícitamente: net/http.Server con valores cero permite
// que un cliente lento mantenga conexiones abiertas indefinidamente
// (slowloris attack). Estos valores son conservadores pero seguros para una
// API REST que devuelve JSON pequeño.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	// Logger estructurado JSON. Usado por main + middleware de HTTP.
	// slog está en stdlib desde Go 1.21 — sin dependencias.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Modo de Gin configurable por env. Default release: el debug mode emite
	// warnings en logs y expone routing por stdout — no apto para producción.
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Path del catálogo: configurable por env (12-factor).
	dataPath := strings.TrimSpace(os.Getenv("PRODUCTS_FILE"))
	if dataPath == "" {
		dataPath = "testdata/products.json"
	}

	// Validación defensiva del path. Filtra typos/configs malas que apunten a
	// archivos no deseados. No es boundary security — es defense-in-depth.
	if err := validateProductsPath(dataPath); err != nil {
		logger.Error("invalid PRODUCTS_FILE", slog.String("path", dataPath), slog.Any("error", err))
		os.Exit(1)
	}

	// Carga del catálogo. Fail-fast: un servicio sin datos no debería arrancar.
	repo, err := persistence.NewJSONRepository(dataPath)
	if err != nil {
		logger.Error("loading products failed", slog.String("path", dataPath), slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("products loaded", slog.String("path", dataPath))

	// Use cases — todos toman el mismo repo.
	compareUC := application.NewCompareProductsUseCase(repo)
	listUC := application.NewListProductsUseCase(repo)
	getUC := application.NewGetProductUseCase(repo)
	categoriesUC := application.NewListCategoriesUseCase(repo)

	handler := httpadapter.NewProductHandler(compareUC, listUC, getUC, categoriesUC)
	router := httpadapter.NewRouter(handler, logger)

	srv := &http.Server{
		Addr:              ":" + portFromEnv(),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Server en goroutine para no bloquear la espera de señales.
	// El error http.ErrServerClosed es esperado al hacer Shutdown — no es fallo.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Graceful shutdown: esperamos SIGINT (Ctrl+C dev) o SIGTERM (kubectl, systemd).
	// Le damos hasta shutdownTimeout para drenar requests en vuelo.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		logger.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	case sig := <-stop:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed, forcing close", slog.Any("error", err))
		_ = srv.Close()
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}

// portFromEnv devuelve el puerto desde la env var PORT o 8080 por default.
func portFromEnv() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

// validateProductsPath sanitiza el path del catálogo.
//
// No es defensa contra atacante con control de env vars (ya tendría algo
// equivalente a RCE). Es defense-in-depth contra typos y configs erradas:
// rechaza extensiones que no sean .json y segmentos de path traversal.
func validateProductsPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("path is empty")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		return errors.New("path must end in .json")
	}
	if strings.HasPrefix(filepath.Clean(path), "..") {
		return errors.New("path contains traversal segments")
	}
	return nil
}
