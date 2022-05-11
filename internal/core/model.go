package core

import (
	"fmt"
	"time"
)

type IPList struct {
	IPS []*IP
}

type IP struct {
	Addr            string // IP地址 IP:PORT
	Type            IPType // IP类型 比如 http，https等等
	Speed           int    // IP速度 单位毫米
	Source          string // IP来源
	LastVisit       int    // 最后一次查看（使用）此IP的时间
	LastSpeedAt     int    // 最后一次速度检查时间
	CreatedAt       int    // 创建时间
	SpeedChecking   bool   // IP测速中
	SpeedErrorCount int    // 测速错误次数
}

func NewIP() *IP {
	return &IP{}
}

func (i *IP) Key() string {
	return fmt.Sprintf("%s-%s", i.Addr, i.Type)
}

func (i *IP) String() string {
	return fmt.Sprintf("addr:%s type:%s speed:%d source:%s lastVisit:%d lastSpeedAt:%d speedErrorCount:%d", i.Addr, string(i.Type), i.Speed, i.Source, i.LastVisit, i.LastSpeedAt, i.SpeedErrorCount)
}

type everyScheduler struct {
	Interval time.Duration
}

func (s *everyScheduler) Next(prev time.Time) time.Time {
	return prev.Add(s.Interval)
}
