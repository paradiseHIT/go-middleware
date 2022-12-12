package app

import (
	"encoding/json"
	"time"
)

type AsyncJob struct {
	JobId         uint32    `gorm:"primary_key" json:"id,omitempty"`
	RequestId     string    `gorm:"Column:request_id" json:"request_id,omitempty"`
	State         uint32    `gorm:"Column:state" json:"state"`
	CreateTime    time.Time `gorm:"Column:create_time" json:"create_time"`
	UpdateTime    time.Time `gorm:"Column:update_time" json:"update_time"`
	Debug         bool      `gorm:"Column:debug" json:"debug,omitempty" query:"debug" form:"debug" xml:"debug" default:"false"`
	Prompt        string    `gorm:"Column:prompt" json:"prompt" query:"prompt" form:"prompt" xml:"prompt"`
	RandomSeed    int       `gorm:"Column:random_seed" json:"random_seed" default:"42"`
	GuidanceScale float32   `gorm:"Column:guidance_scale" json:"guidance_scale" default:"9.0"`
	Width         int       `gorm:"Column:width" json:"width" default:"768"`
	Height        int       `gorm:"Column:height" json:"height" default:"768"`
	Result        string    `gorm:"type:text,omitempty"`
}

type JobRequest struct {
	JobId         uint32  `json:"job_id" query:"job_id" form:"job_id" xml:"job_id"`
	Debug         bool    `json:"debug,omitempty" query:"debug" form:"debug" xml:"debug" default:"false"`
	Prompt        string  `json:"prompt" query:"prompt" form:"prompt" xml:"prompt"`
	RandomSeed    int     `json:"random_seed" default:"42"`
	GuidanceScale float32 `json:"guidance_scale"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
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
