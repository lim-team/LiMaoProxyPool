package core

import "time"

type Options struct {
	IPListenGoroutine   int           // 监听IP的协程数量
	SpeedCheckInterval  time.Duration // 速度检查间隔
	SpeedCheckGoroutine int           // 速度检测协程数量
	SpeedMax            int
}

func NewOptions() *Options {

	return &Options{
		IPListenGoroutine:   50,
		SpeedCheckGoroutine: 50,
		SpeedCheckInterval:  time.Minute * 10,
		SpeedMax:            100000,
	}
}
