package app

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var cnt = 0

const (
	JOB_STATE_INIT       = 0
	JOB_STATE_PROCESSING = 1
	JOB_STATE_SUCCESS    = 2
	JOB_STATE_FAILED     = 3

	PARAMETER_ERROR         = "PARAMETER_ERROR"
	INTERNAL_ERROR          = "INTERNAL_ERROR"
	POST_REQUEST_ERROR      = "POST_REQUEST_ERROR"
	INTERNAL_RESPONSE_ERROR = "INTERNAL_RESPONSE_ERROR"
	CODE_OK                 = "OK"
)

type Server struct {
	server_address             string
	stablediffusion_timeout    time.Duration
	stablediffusion_requestURL string
	stablediffusion_method     string
	client                     *http.Client
	req_glob                   *http.Request
	q                          *Queue
	db                         *gorm.DB
}

func (s *Server) MakeResponse(requestId, code, message string) *Response {
	response := Response{
		RequestId: requestId,
		Code:      code,
		Message:   message,
	}
	return &response
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func (s *Server) InitServer(config_file string) {
	s.InitConfig(config_file)
	s.stablediffusion_timeout = viper.GetDuration("stablediffusion.timeout")
	s.stablediffusion_requestURL = viper.GetString("requestURL")
	s.stablediffusion_method = viper.GetString("stablediffusion.method")
	s.server_address = viper.GetString("address")

	//reuse the http client to get high performance
	s.client = &http.Client{Timeout: time.Duration(s.stablediffusion_timeout) * time.Second}

	var err error
	s.req_glob, err = http.NewRequest(s.stablediffusion_method, s.stablediffusion_requestURL, nil)
	if err != nil {
		usageAndExit(err.Error())
	}

	// set content-type
	header := make(http.Header)
	header.Set("Content-Type", viper.GetString("stablediffusion.content_type"))
	s.req_glob.Header = header

	s.db, _ = s.DBConnect(viper.GetString("DBAddress"))
	s.InitQueue()
	s.db.AutoMigrate(&AsyncJob{})
}

func (s *Server) StartServer() {
	go s.ProcessBackend()
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "Authorization"},
	}))

	e.POST("/process", s.Process)
	e.POST("/query", s.Query)
	err := e.Start(s.server_address)
	if err != nil {
		glog.Fatal(err.Error())
	}
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func (s *Server) cloneRequest(r *http.Request, body []byte) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	if len(body) > 0 {
		r2.Body = ioutil.NopCloser(bytes.NewReader(body))
	}
	r2.ContentLength = int64(len(body))
	return r2
}

func (s *Server) Process(c echo.Context) error {
	cnt += 1
	requestId := s.GenerateReqId()
	jobReq := new(JobRequest)
	err := c.Bind(jobReq)
	if err != nil {
		glog.Infoln("Bind error", err)
		return c.JSON(http.StatusBadRequest, s.MakeResponse(requestId, PARAMETER_ERROR, err.Error()))
	}
	glog.Infof("request detail:%v\n", jobReq)

	post_data := &bytes.Buffer{}
	encoder := json.NewEncoder(post_data)
	if err := encoder.Encode(jobReq); err != nil {
		return c.JSON(http.StatusBadRequest, s.MakeResponse(requestId, PARAMETER_ERROR, err.Error()))
	}

	now := time.Now()
	job := AsyncJob{
		State:         JOB_STATE_INIT,
		RequestId:     requestId,
		CreateTime:    now,
		UpdateTime:    now,
		Debug:         jobReq.Debug,
		Prompt:        jobReq.Prompt,
		RandomSeed:    jobReq.RandomSeed,
		Steps:         jobReq.Steps,
		Width:         jobReq.Width,
		Height:        jobReq.Height,
		GuidanceScale: jobReq.GuidanceScale,
		NegPrompt:     jobReq.NegPrompt,
		NIter:         jobReq.NIter,
		Sampler:       jobReq.Sampler,
	}
	err = s.db.Create(&job).Error
	if err != nil {
		glog.Infoln(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, requestId, err))
		return c.JSON(http.StatusBadRequest, s.MakeResponse(requestId, INTERNAL_ERROR, err.Error()))
	}
	s.q.Push(job)
	response := s.MakeResponse(requestId, CODE_OK, "success")
	response.Data = make(map[string]interface{})
	response.Data["job_id"] = job.JobId
	response.Data["queue_len"] = s.q.Len()
	return c.JSON(http.StatusOK, response)
}

//func ProcessBackend(requestId string, now time.Time, job *AsyncJob, post_data *bytes.Buffer) {

func (s *Server) ProcessBackend() {
	for {
		if s.q.Len() == 0 {
			time.Sleep(1 * time.Second)
			continue
		}
		var job AsyncJob
		job, ok := s.q.Pop().(AsyncJob)
		if !ok {
			glog.Infoln("job in queue is not type AsyncJob")
			continue
		}
		job_detail := make(map[string]interface{})
		job_detail["prompt"] = job.Prompt
		job_detail["debug"] = job.Debug
		job_detail["random_seed"] = job.RandomSeed
		job_detail["steps"] = job.Steps
		job_detail["width"] = job.Width
		job_detail["height"] = job.Height
		job_detail["guidance_scale"] = job.GuidanceScale
		job_detail["negative_prompt"] = job.NegPrompt
		job_detail["n_iter"] = job.NIter
		job_detail["sampler"] = job.Sampler
		post_data, err := json.Marshal(job_detail)
		job.State = JOB_STATE_PROCESSING
		job.UpdateTime = time.Now()
		if err := s.db.Updates(job).Error; err != nil {
			glog.Infoln(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, job.RequestId, err))
			continue
		}
		if s.req_glob == nil {
			panic("glob req is null")
		}
		req := s.cloneRequest(s.req_glob, post_data)
		glog.Infof("post_data is %s\n", post_data)
		res, err := s.client.Do(req)
		if err != nil {
			job.State = JOB_STATE_FAILED
			job.UpdateTime = time.Now()
			job.Result = fmt.Sprintf(`{"error":"%s"}`, POST_REQUEST_ERROR)
			if dberr := s.db.Updates(job).Error; dberr != nil {
				glog.Infoln(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, job.RequestId, dberr))
			}
			glog.Infof("[%s]:error post http request %s to %s: %s", post_data, job.RequestId, req.URL, err.Error())
			continue
		} else {
			respBytes, err := ioutil.ReadAll(res.Body)
			if err != nil {
				glog.Infof("[%s]:error making http request to %s: %s", job.RequestId, req.URL, err.Error())
				job.State = JOB_STATE_FAILED
				job.UpdateTime = time.Now()
				job.Result = fmt.Sprintf(`{"error":"%s"}`, INTERNAL_RESPONSE_ERROR)
				if dberr := s.db.Updates(job).Error; dberr != nil {
					glog.Infoln(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, job.RequestId, dberr))
				}
				continue
			}
			job.State = JOB_STATE_SUCCESS
			job.UpdateTime = time.Now()
			job.Result = string(respBytes)
			if dberr := s.db.Updates(job).Error; dberr != nil {
				glog.Infoln(fmt.Sprintf("state=%d&requestId=%s&error=%+v", job.State, job.RequestId, dberr))
			}
			glog.Infoln(fmt.Sprintf("state=%d&requestId=%s", job.State, job.RequestId))
		}
	}
}

func (s *Server) GenerateReqId() string {
	return uuid.New().String()
}

//mysql dsn:
//root:123456@tcp(127.0.01:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local

func (s *Server) DBConnect(dbAddress string) (*gorm.DB, error) {
	if idb, err := gorm.Open(mysql.Open(dbAddress), &gorm.Config{Logger: logger.Default.LogMode(logger.Error)}); err == nil {
		return idb, nil
	} else {
		if err != nil {
			log.Fatal(err)
		}
		return nil, fmt.Errorf("Failed to connect database: %v", err.Error())
	}
}

func (s *Server) Query(c echo.Context) error {
	requestId := s.GenerateReqId()
	req := new(JobRequest)
	err := c.Bind(req)
	if err != nil {
		glog.Infoln("Bind error", err)
		return c.JSON(http.StatusBadRequest, s.MakeResponse(requestId, PARAMETER_ERROR, err.Error()))
	}
	var job AsyncJob
	if err := s.db.First(&job, "job_id=?", req.JobId).Error; err != nil {
		message := fmt.Sprintf("job_id %d not exists", req.JobId)
		glog.Infof("requestId:%+v, job_id:%+v, err:%+v", requestId, req.JobId, err)
		return c.JSON(http.StatusBadRequest, s.MakeResponse(requestId, PARAMETER_ERROR, message))
	}
	response := s.MakeResponse(requestId, CODE_OK, "success")
	response.Data = make(map[string]interface{})
	response.Data["result"] = job
	if job.State == JOB_STATE_PROCESSING || job.State == JOB_STATE_SUCCESS || job.State == JOB_STATE_FAILED {
		response.Data["queue_len"] = 0
	} else {
		var count int64
		if err := s.db.Model(&job).Where("create_time < ? and state = 0", job.CreateTime).Count(&count).Error; err != nil {
			message := "get job count faild"
			glog.Infof("requestId:%+v, job_id:%+v, create_time:%+v, state:%+v, err:%+v", requestId, job.JobId, job.CreateTime, job.State, err)
			return c.JSON(http.StatusBadRequest, s.MakeResponse(requestId, INTERNAL_ERROR, message))
		}
		response.Data["queue_len"] = count
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) InitQueue() {
	s.q = new(Queue)
	var jobs []AsyncJob
	if err := s.db.Where("state = ?", JOB_STATE_INIT).Find(&jobs).Error; err != nil {
		message := "get unprocessed job failed"
		glog.Infoln(message)
		return
	}
	for _, v := range jobs {
		s.q.Push(v)
		glog.Infof("load old job %d in to queue\n", v.JobId)
	}
}

func (s *Server) InitConfig(config_file string) {
	viper.SetConfigFile(config_file)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			panic("config file not found [" + config_file + "]")
		} else {
			// Config file was found but another error was produced
			glog.Fatal("Fatal error config file: ")
		}
	}
	glog.Infoln("backend service URL is :[" + viper.GetString("requestURL") + "]")
}
