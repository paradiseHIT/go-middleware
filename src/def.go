package main

import (
  "time"
  "encoding/json"
)

const (
  JOB_STATE_INIT    = 0
  JOB_STATE_PROCESSING    = 1
  JOB_STATE_SUCCESS = 2
  JOB_STATE_FAILED  = 3

  PARAMETER_ERROR = "PARAMETER_ERROR"
  INTERNAL_ERROR = "INTERNAL_ERROR"
  POST_REQUEST_ERROR = "POST_REQUEST_ERROR"
  INTERNAL_RESPONSE_ERROR = "INTERNAL_RESPONSE_ERROR"
  CODE_OK = "OK"
)

type AsyncJob struct {
  JobId uint32 `gorm:"primary_key" json:"id,omitempty"`
  RequestId string `gorm:"Column:request_id" json:"request_id,omitempty"`
  State uint32 `gorm:"Column:state" json:"state"`
  CreateTime time.Time `gorm:"Column:create_time" json:"create_time"`
  UpdateTime time.Time `gorm:"Column:update_time" json:"update_time"`
  Debug string `json:"debug,omitempty" query:"debug" form:"debug" xml:"debug" default:"false"`
  Prompt string `json:"prompt" query:"prompt" form:"prompt" xml:"prompt"`
  Result string `gorm:"type:text,omitempty"`
}

type Request struct {
  JobId uint32 `json:"job_id" query:"job_id" form:"job_id" xml:"job_id"`
  Debug string `json:"debug,omitempty" query:"debug" form:"debug" xml:"debug" default:"false"`
  Prompt string `json:"prompt" query:"prompt" form:"prompt" xml:"prompt"`
}

type Response struct {
  RequestId string                 `json:"request_id,omitempty"`
  Code      string                 `json:"code"`
  Message   string                 `json:"message,omitempty"`
  Data      map[string]interface{} `json:"data,omitempty"`
}

func (r Response) ToString() string {
  str, _ := json.Marshal(r)
  return string(str)
}

func MakeResponse(requestId, code, message string) *Response {
  response := Response{
    RequestId: requestId,
    Code:      code,
    Message:   message,
  }
  return &response
}
