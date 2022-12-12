// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"flag"
	"middleware/api-service/app"

	"github.com/golang/glog"
	"github.com/labstack/echo/v4"
)

var config_file *string
var s *app.Server

func ProcessRequest(ctx echo.Context) error {
	return s.Process(ctx)
}

func Query(ctx echo.Context) error {
	return s.Query(ctx)
}

func init() {
	config_file = flag.String("config", "", "config file path")
}

func main() {
	flag.Parse()
	defer glog.Flush()
	if *config_file == "" {
		panic("config_file [" + *config_file + "] is not set")
	}
	glog.Infoln("config file is ", *config_file)
	s = new(app.Server)
	s.InitServer(*config_file)
	s.StartServer()
}
