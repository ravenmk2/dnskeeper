package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
	"github.com/ravenmk2/dnskeeper/internal/config"
	"github.com/ravenmk2/dnskeeper/internal/envelope"
	"github.com/ravenmk2/dnskeeper/internal/handler"
	mw "github.com/ravenmk2/dnskeeper/internal/middleware"
	"github.com/ravenmk2/dnskeeper/internal/jwt"
	"github.com/ravenmk2/dnskeeper/internal/service"
	"github.com/ravenmk2/dnskeeper/internal/store"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type App struct {
	echo   *echo.Echo
	etcd   *clientv3.Client
	store  store.Store
	logger *logrus.Logger
	listen string
}

type customValidator struct {
	v *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	return cv.v.Struct(i)
}

func New(cfg *config.Config, logger *logrus.Logger) (*App, error) {
	etcdCli, err := newEtcdClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create etcd client: %w", err)
	}
	failed := true
	defer func() {
		if failed {
			etcdCli.Close()
		}
	}()
	s := store.New(etcdCli, cfg.CoreDNS.Path)

	accessTTL, err := cfg.JWT.ParseAccessTTL()
	if err != nil {
		return nil, err
	}
	refreshTTL, err := cfg.JWT.ParseRefreshTTL()
	if err != nil {
		return nil, err
	}
	jwtMgr := jwt.NewManager(cfg.JWT.Secret, accessTTL, refreshTTL)

	svcs := service.NewServices(s, jwtMgr)
	h := handler.New(svcs, s)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = &customValidator{v: validator.New()}
	e.HTTPErrorHandler = errorHandler

	e.Use(mw.RequestID)
	e.Use(mw.Logger(logger))
	e.Use(mw.Recover(logger))

	registerRoutes(e, h, jwtMgr)

	failed = false
	return &App{
		echo:   e,
		etcd:   etcdCli,
		store:  s,
		logger: logger,
		listen: cfg.Server.Listen,
	}, nil
}

func registerRoutes(e *echo.Echo, h *handler.Handlers, jwtMgr *jwt.Manager) {
	e.GET("/api/health", h.Health)
	e.POST("/api/health", h.Health)
	e.POST("/api/auth/login", h.Login)
	e.POST("/api/auth/refresh", h.Refresh)

	prot := e.Group("/api", mw.Auth(jwtMgr))
	prot.POST("/me", h.Me)
	prot.POST("/me/change-password", h.ChangePassword)
	prot.POST("/dns/zone/list", h.ListZones)
	prot.POST("/dns/zone/get", h.GetZone)
	prot.POST("/dns/domain/list", h.ListDomains)
	prot.POST("/dns/domain/get", h.GetDomain)
	prot.POST("/dns/record/list", h.ListRecords)
	prot.POST("/dns/record/get", h.GetRecord)
	prot.POST("/dns/record/create", h.CreateRecord)
	prot.POST("/dns/record/update", h.UpdateRecord)
	prot.POST("/dns/record/delete", h.DeleteRecord)

	adm := e.Group("/api", mw.Auth(jwtMgr), mw.RequireAdmin)
	adm.POST("/user/list", h.ListUsers)
	adm.POST("/user/create", h.CreateUser)
	adm.POST("/user/update", h.UpdateUser)
	adm.POST("/user/delete", h.DeleteUser)
	adm.POST("/dns/zone/create", h.CreateZone)
	adm.POST("/dns/zone/update", h.UpdateZone)
	adm.POST("/dns/zone/delete", h.DeleteZone)
	adm.POST("/dns/domain/create", h.CreateDomain)
	adm.POST("/dns/domain/update", h.UpdateDomain)
	adm.POST("/dns/domain/delete", h.DeleteDomain)
}

func errorHandler(err error, c echo.Context) {
	if ae, ok := apperr.As(err); ok {
		envelope.Error(c, ae.HTTPCode, ae)
		return
	}
	if he, ok := err.(*echo.HTTPError); ok {
		code := he.Code
		if code == 0 {
			code = http.StatusInternalServerError
		}
		var ae *apperr.AppError
		switch code {
		case http.StatusNotFound:
			ae = apperr.New("NOT_FOUND", "not found", code)
		case http.StatusMethodNotAllowed:
			ae = apperr.New("METHOD_NOT_ALLOWED", "method not allowed", code)
		default:
			ae = apperr.InternalError
		}
		envelope.Error(c, code, ae)
		return
	}
	envelope.Error(c, http.StatusInternalServerError, apperr.InternalError)
}

func (a *App) SeedAdmin(ctx context.Context) error {
	_, err := a.store.GetUser(ctx, "admin")
	if err == nil {
		return nil
	}
	if !errors.Is(err, apperr.UserNotFound) {
		return err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.logger.Warn("seeding default admin user with password 'admin123' — change it immediately via /api/me/change-password")
	ts := time.Now().UTC().Format(time.RFC3339)
	user := &store.User{
		ID:        "admin",
		Username:  "admin",
		Password:  string(hashed),
		UserType:  "admin",
		Builtin:   true,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	data, err := store.MarshalUser(user)
	if err != nil {
		return err
	}
	return a.store.Put(ctx, store.UserKey("admin"), data)
}

func (a *App) Run() error {
	return a.echo.Start(a.listen)
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := a.echo.Shutdown(ctx); err != nil {
		return err
	}
	return a.etcd.Close()
}

func newEtcdClient(cfg *config.Config) (*clientv3.Client, error) {
	ec := clientv3.Config{
		Endpoints: cfg.Etcd.Endpoints,
		Username:  cfg.Etcd.Username,
		Password:  cfg.Etcd.Password,
	}
	if cfg.Etcd.TLS() {
		tlsCfg, err := loadTLSConfig(cfg.Etcd)
		if err != nil {
			return nil, fmt.Errorf("load tls: %w", err)
		}
		ec.TLS = tlsCfg
	}
	cli, err := clientv3.New(ec)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func loadTLSConfig(c config.EtcdConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(c.Cert, c.Key)
	if err != nil {
		return nil, err
	}
	caData, err := os.ReadFile(c.CA)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to parse CA certificates from %s", c.CA)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}, nil
}
