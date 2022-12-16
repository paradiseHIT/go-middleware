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
	NegPrompt     string    `gorm:"Column:negative_prompt" json:"negative_prompt" query:"negative_prompt" form:"negative_prompt" xml:"negative_prompt"`
	RandomSeed    int       `gorm:"Column:random_seed" json:"random_seed" default:"42"`
	GuidanceScale float32   `gorm:"Column:guidance_scale" json:"guidance_scale" default:"9.0"`
	Width         int       `gorm:"Column:width" json:"width" default:"768"`
	Height        int       `gorm:"Column:height" json:"height" default:"768"`
	Steps         int       `gorm:"Column:steps" json:"steps" default:"768"`
	NIter         int       `gorm:"Column:n_iter" json:"n_iter" default:"3"`
	Sampler       string    `gorm:"Column:sampler" json:"sampler" form:"sampler" default:"plms"`
	Result        string    `gorm:"type:text,omitempty"`
}

type JobRequest struct {
	JobId         uint32  `json:"job_id" query:"job_id" form:"job_id" xml:"job_id"`
	Debug         bool    `json:"debug,omitempty" query:"debug" form:"debug" xml:"debug" default:"false"`
	Prompt        string  `json:"prompt" query:"prompt" form:"prompt" xml:"prompt"`
	NegPrompt     string  `json:"negative_prompt" query:"negative_prompt" form:"negative_prompt" xml:"negative_prompt"`
	RandomSeed    int     `json:"random_seed" form:"random_seed" default:"42"`
	GuidanceScale float32 `json:"guidance_scale" form:"guidance_scale" `
	Width         int     `json:"width" form:"width" `
	Height        int     `json:"height" form:"height" `
	Steps         int     `json:"steps" form:"steps" default:"50"`
	NIter         int     `json:"n_iter" form:"n_iter" default:"3"`
	Sampler       string  `json:"sampler" form:"sampler" default:"plms"`
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
