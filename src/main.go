// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
  "flag"
  "time"
  "fmt"
  "bytes"
  "log"
  "io/ioutil"
  "encoding/json"
  "net/http"
  "github.com/google/uuid"
  "github.com/spf13/viper"
  "github.com/labstack/echo/v4"
  "github.com/labstack/echo/v4/middleware"
  "gorm.io/gorm/logger"
  "gorm.io/gorm"
  "gorm.io/driver/mysql"
)
var q *Queue
var cnt = 0
var db *gorm.DB

func Process(c echo.Context) error {
  cnt += 1
  requestId := GenerateReqId()
  req := new(Request)
  err := c.Bind(req); if err != nil {
    log.Println("Bind error", err)
    return c.JSON(http.StatusBadRequest, MakeResponse(requestId, PARAMETER_ERROR, err.Error()))
  }
  //req.Debug = "debug"

  post_data := &bytes.Buffer{}
  encoder := json.NewEncoder(post_data)
  if err := encoder.Encode(req); err != nil {
    return c.JSON(http.StatusBadRequest, MakeResponse(requestId, PARAMETER_ERROR, err.Error()))
  }

  now := time.Now()
  job := AsyncJob{
    State: JOB_STATE_INIT,
    RequestId: requestId,
    CreateTime: now,
    UpdateTime: now,
    Debug: req.Debug,
    Prompt: req.Prompt,
  }
  err = db.Create(&job).Error
  if err != nil {
    log.Println(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, requestId, err))
    return c.JSON(http.StatusBadRequest, MakeResponse(requestId, INTERNAL_ERROR, err.Error()))
  }
  q.Push(job)
  response := MakeResponse(requestId, CODE_OK, "success")
  response.Data = make(map[string]interface{})
  response.Data["job_id"] = job.JobId
  response.Data["queue_len"] = q.Len()
  return c.JSON(http.StatusOK, response.ToString())
}

//func ProcessBackend(requestId string, now time.Time, job *AsyncJob, post_data *bytes.Buffer) {

func ProcessBackend() {
  for {
    if q.Len() == 0 {
      time.Sleep(1*time.Second)
      continue
    }
    var job AsyncJob
    job, ok := q.Pop().(AsyncJob)
    if !ok {
      log.Println("job in queue is not type AsyncJob")
      continue
    }

    req := make(map[string]interface{})
    req["prompt"] = job.Prompt
    post_data, err := json.Marshal(req)
    if err != nil {
      log.Println("json Marshal failed", req)
      continue
    }
    job.State = JOB_STATE_PROCESSING
    job.UpdateTime = time.Now()
    requestURL := viper.GetString("requestURL")
    if err := db.Updates(job).Error; err != nil {
      log.Println(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, job.RequestId, err))
      continue
    }
    res, err := http.Post(requestURL, "application/json", bytes.NewBuffer(post_data))
    if err != nil {
      job.State = JOB_STATE_FAILED
      job.UpdateTime = time.Now()
      job.Result = fmt.Sprintf(`{"error":"%s"}`, POST_REQUEST_ERROR)
      if dberr := db.Updates(job).Error; dberr != nil {
        log.Println(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, job.RequestId, dberr))
      }
      log.Printf("[%s]:error post http request %s to %s: %s",post_data, job.RequestId, requestURL, err.Error())
      continue
    } else {
      respBytes, err := ioutil.ReadAll(res.Body)
      if err != nil {
        log.Printf("[%s]:error making http request to %s: %s",job.RequestId, requestURL, err.Error())
        job.State = JOB_STATE_FAILED
        job.UpdateTime = time.Now()
        job.Result = fmt.Sprintf(`{"error":"%s"}`, INTERNAL_RESPONSE_ERROR)
        if dberr := db.Updates(job).Error; dberr != nil {
          log.Println(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, job.RequestId, dberr))
        }
        continue
      }
      job.State = JOB_STATE_SUCCESS
      job.UpdateTime = time.Now()
      job.Result = string(respBytes)
      if dberr := db.Updates(job).Error; dberr != nil {
        log.Println(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, job.RequestId, dberr))
      }
    }
  }
}

func GenerateReqId() string {
  return uuid.New().String()
}

func InitConfig(filename string) {
  viper.SetConfigFile(filename)
  if err := viper.ReadInConfig(); err != nil {
    if _, ok := err.(viper.ConfigFileNotFoundError); ok {
      // Config file not found; ignore error if desired
      panic("config file not found [" + filename + "]")
    } else {
      // Config file was found but another error was produced
      log.Fatal("Fatal error config file: ")
    }
  }
  log.Println("backend service URL is :[" + viper.GetString("requestURL") + "]")
}
//mysql dsn:
//root:123456@tcp(127.0.01:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local

func DBConnect(dbAddress string) (*gorm.DB, error) {
  if idb, err := gorm.Open(mysql.Open(dbAddress), &gorm.Config{Logger: logger.Default.LogMode(logger.Error)}); err == nil {
    return idb, nil
  } else {
    if err != nil {
      log.Fatal(err)
    }
    return nil, fmt.Errorf("Failed to connect database: %v", err.Error())
  }
}

func Query(c echo.Context) error {
  requestId := GenerateReqId()
  req := new(Request)
  err := c.Bind(req); if err != nil {
    log.Println("Bind error", err)
    return c.JSON(http.StatusBadRequest, MakeResponse(requestId, PARAMETER_ERROR, err.Error()))
  }
  var job AsyncJob
  if err := db.First(&job, "job_id=?", req.JobId).Error; err != nil {
    message := fmt.Sprintf("job_id %d not exists", req.JobId)
    log.Printf("requestId:%+v, job_id:%+v, err:%+v", requestId, req.JobId, err)
    return c.JSON(http.StatusBadRequest, MakeResponse(requestId, PARAMETER_ERROR, message))
  }
  response := MakeResponse(requestId, CODE_OK, "success")
  response.Data = make(map[string]interface{})
  response.Data["result"] = job
  if job.State == JOB_STATE_PROCESSING || job.State == JOB_STATE_SUCCESS || job.State == JOB_STATE_FAILED {
    response.Data["queue_len"] = 0
  } else {
    var count int64
    if err := db.Model(&job).Where("create_time < ? and state = 0",  job.CreateTime).Count(&count).Error; err != nil {
      message := "get job count faild"
      log.Printf("requestId:%+v, job_id:%+v, create_time:%+v, state:%+v, err:%+v", requestId, job.JobId, job.CreateTime, job.State, err)
      return c.JSON(http.StatusBadRequest, MakeResponse(requestId, INTERNAL_ERROR, message))
    }
    response.Data["queue_len"] = count
  }
  return c.JSON(http.StatusOK, response)
}

func InitQueue() *Queue {
  q := new(Queue)
  var jobs []AsyncJob
  if err := db.Where("state = ?", JOB_STATE_INIT).Find(&jobs).Error; err != nil {
    message := "get unprocessed job failed"
    log.Println(message)
    return nil
  }
  for _, v := range jobs {
    q.Push(v)
    log.Println("%d", v.JobId)
  }
  return q
}
var config_file *string
func init() {
  config_file = flag.String("config", "", "config file path")
}

func main() {
  flag.Parse()
  if *config_file == "" {
    panic("config_file [" + *config_file + "] is not set") 
  }
  log.Println("config file is ", *config_file)
  InitConfig(*config_file)
  e := echo.New()
  e.HideBanner = true
  e.Use(middleware.Recover())
  e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"*"},
    AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "Authorization"},
  }))
  db, _ = DBConnect(viper.GetString("DBAddress"))
  q = InitQueue()
  db.AutoMigrate(&AsyncJob{})
  go ProcessBackend()
  e.POST("/process", Process)
  e.POST("/query", Query)
  //e.GET("/list", List)
  e.Logger.Fatal(e.Start(viper.GetString("address")))
}