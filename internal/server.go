package internal

import (
	_ "ABIevm/docs"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type Server struct {
	e *echo.Echo
}

func NewServer() *Server {
	s := &Server{
		e: echo.New(),
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.e.GET("/swagger/*", echoSwagger.WrapHandler)

	api := NewHandler()
	s.e.GET("/reservesAave/:network", api.HandleGetAaveReserves)

	s.e.GET("/reservesCurve/:network", api.HandleGetCurveReserves)

	s.e.GET("/reservesStakedao/:network", api.HandleGetStakeDaoReserves)
}

func (s *Server) Start() {
	s.e.Logger.Fatal(s.e.Start(":8081"))
}
