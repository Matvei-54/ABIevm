package internal

import (
	_ "ABIevm/docs"
	"ABIevm/internal/service/stakedao"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type Server struct {
	e *echo.Echo
}

func NewServer() *Server { return NewServerWithRegistry(nil) }

func NewServerWithRegistry(store stakedao.RegistryStore) *Server {
	s := &Server{
		e: echo.New(),
	}

	s.routes(NewHandlerWithRegistry(store))
	return s
}

func (s *Server) routes(api *Handler) {
	s.e.GET("/swagger/*", echoSwagger.WrapHandler)

	s.e.GET("/reservesAave/:network", api.HandleGetAaveReserves)

	s.e.GET("/reservesCurve/:network", api.HandleGetCurveReserves)

	s.e.GET("/reservesStakedao/:network", api.HandleGetStakeDaoReserves)
}

func (s *Server) Start() {
	if err := s.Run(context.Background()); err != nil {
		s.e.Logger.Fatal(err)
	}
}

// Run lets main close the Postgres pool after Echo finishes active requests.
func (s *Server) Run(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- s.e.Start(":8081") }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := s.e.Shutdown(shutdownCtx)
		if err != nil {
			_ = s.e.Close()
		}
		<-done
		return err
	}
}
