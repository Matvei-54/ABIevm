package main

import (
	in "ABIevm/internal"
	//"log"
	//"os"
	//
	//"github.com/labstack/echo/v4"
)

func main() {

	server := in.NewServer()
	server.Start()
}
