package db

import (
    "gorm.io/gorm"
    "time"
    )

type AsyncJob struct {
  JobId uint32 `gorm:"primary_key" json:"id,omitempty"`
  CreateTime time.Time `gorm:"Column:create_time" json:"create_time"`
}
