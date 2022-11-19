// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
  //"fmt"
  "bytes"
	"log"
  "io/ioutil"
  "encoding/json"
	"net/http"
  "github.com/google/uuid"
	"github.com/spf13/viper"
  "github.com/labstack/echo/v4"
  "gorm.io/gorm/logger"
  "gorm.io/gorm"
)
var cnt = 0
type Request struct {
  Debug string `json:"debug,omitempty" query:"debug" form:"debug" xml:"debug" default:"false"`
  Prompt string `json:"prompt" query:"prompt" form:"prompt" xml:"prompt"`
}
type Result struct {
  ret string `json:"ret,omitempty"`
}

type Response struct {
  RequestId string                 `json:"request_id,omitempty"`
  Code      string                 `json:"code"`
  Message   string                 `json:"message,omitempty"`
  Data      map[string]interface{} `json:"data,omitempty"`
}

func MakeResponse(requestId, code, message string) *Response {
  response := Response{
    RequestId: requestId,
    Code:      code,
    Message:   message,
  }
  return &response
}

const(
  PARAMETER_ERROR = "PARAMETER_ERROR"
)

func Process(c echo.Context) error {
  cnt += 1
  requestId := GenerateReqId()
  req := new(Request)
  err := c.Bind(req); if err != nil {
    log.Println("Bind error", err)
    return c.JSON(http.StatusBadRequest, MakeResponse("", PARAMETER_ERROR, err.Error()))
  }
  log.Println(req)
  //req.Debug = "debug"

  log.Println("req:", req)
  post_data := &bytes.Buffer{}
  encoder := json.NewEncoder(post_data)
  if err := encoder.Encode(req); err != nil {
    return c.JSON(http.StatusBadRequest, MakeResponse("", "encode error", err.Error()))
  }
  log.Println("post data:", post_data.String())
  requestURL := viper.GetString("requestURL")
  res, err := http.Post(requestURL, "application/json", post_data)
  if err != nil {
    log.Printf("error making http request to %s: %s,", requestURL, err)
    log.Printf("res : %d\n", res.StatusCode)
		log.Printf("%d, %s, [%s]\n", cnt, requestId, err.Error())
    return c.JSON(http.StatusBadRequest, MakeResponse("", "POST error", err.Error()))
  } else {
    respBytes, err2 := ioutil.ReadAll(res.Body)
    if err2 != nil {
			log.Printf("%d, %s, [%s]\n", cnt, requestId, err2.Error())
      return c.JSON(http.StatusBadRequest, MakeResponse("", "ReadAll error", err.Error()))
    }
		log.Printf("%d, %s, [%s]\n", cnt, requestId, string(respBytes))
    var dat map[string]interface{}
    if err := json.Unmarshal(respBytes, &dat) ; err != nil {
      panic(err)
    }
    log.Println("dat", dat)
    return c.JSON(http.StatusOK, MakeResponse("", "successful process", dat["ret"].(string)))
    //encoder := json.NewEncoder(respBytes)
    //if err := encoder.Encode(Result); err != nil {
    //  return c.JSON(http.StatusBadRequest, MakeResponse("", "encode result error", err.Error()))
    //}
  }
}

func GenerateReqId() string {
  return uuid.New().String()
}

func InitConfig(path string, filename string) {
  viper.AddConfigPath(".")
  viper.SetConfigName("config")
  if err := viper.ReadInConfig(); err != nil {
    if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Println("config file not found")
		} else {
			// Config file was found but another error was produced
			log.Fatal("Fatal error config file: ")
		}
  }
  log.Println("backend service URL is :[" + viper.GetString("requestURL") + "]")
}
//mysql dsn:
//root:123456@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local

func DBConnect(dbDriver string, dbAddress string) (*gorm.DB, error) {
  log.Debug(fmt.Sprintf("using db %v : %v", dbDriver, dbAddress))
  if db, err := gorm.Open(mysql.Open(dbAddress), &gorm.Config{Logger: logger.Default.LogMode(logger.Error)}); err == nil {
    return db, nil
  } else {
    return nil, fmt.Errorf("Failed to connect database: %v", err.Error())
  }
}

func main() {
  InitConfig(".", "config")
  db, err := DBConnect(viper.GetString("DBDriver"), viper.GetString("DBAddress"))
  if err != nil {
    glog.Fatal(err)
  }
  e := echo.New()
  e.POST("process", Process)
  e.Logger.Fatal(e.Start(":8082"))
}
