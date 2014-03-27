// package log 只是简单的把标准 log 的输出重定向为一个本地的文件, API还是标准的log.
// 为了管理方便, 每天都生成一个新的文件.
// 一个应用只能调用一次
package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const logExt = ".log"

var oncedo sync.Once

// 开始接管标准log的输出, 此后所有用标准log的输出都会输出到此包指定的io.Writer
// logDir 为保存log文件的目录
func Start(logDir string) {
	oncedo.Do(func() { startonce(logDir) })
}

func startonce(logDir string) {
	var currentLogFile *os.File
	var err error
	var date string

	// 首先创建当日文件, 如果存在则追加
	// RFC3339     = "2006-01-02T15:04:05Z07:00"
	date = time.Now().Format(time.RFC3339)[:10]
	currentLogFile, err = os.OpenFile(filepath.Join(logDir, date+logExt), os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		panic(err.Error())
	}
	log.SetOutput(currentLogFile)

	var c = make(chan time.Time)

	go dayTrigger(c)
	go func() {
		var newFile *os.File
		var t time.Time
		var ok bool
		for {
			if t, ok = <-c; ok {
				// RFC3339     = "2006-01-02T15:04:05Z07:00"
				date = t.Format(time.RFC3339)[:10]
				newFile, err = os.OpenFile(filepath.Join(logDir, date+logExt), os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
				if err != nil { // 继续用当前的文件
					fmt.Println(err.Error())
					log.Println(err.Error())
				} else {
					log.SetOutput(newFile)
					if currentLogFile != nil {
						currentLogFile.Close()
					}
					currentLogFile = newFile
				}
			} else { // 管道关闭了,没有必要循环了
				break
			}
		}
	}()
}

func dayTrigger(toChan chan<- time.Time) {
	var t time.Time
	var tk *time.Ticker
	var ch <-chan time.Time

	tk = time.NewTicker(time.Second)
	ch = tk.C
AlignSecond:
	for {
		select {
		case t = <-ch:
			if t.Second() != 0 {
				continue
			}
			tk.Stop()

			if t.Minute() == 0 && t.Hour() == 0 { //新的一天
				toChan <- t
				goto DayLoop
			}

			break AlignSecond
		}
	}

	tk = time.NewTicker(time.Minute)
	ch = tk.C
AlignMinute:
	for {
		select {
		case t = <-ch:
			// now t.Second() == 0
			if t.Minute() != 0 {
				continue
			}
			tk.Stop()

			if t.Hour() == 0 { //新的一天
				toChan <- t
				goto DayLoop
			}

			break AlignMinute
		}
	}

	tk = time.NewTicker(time.Hour)
	ch = tk.C
AlignHour:
	for {
		select {
		case t = <-ch:
			// now t.Second() == 0 and t.Minute() == 0
			if t.Hour() != 0 {
				continue
			}
			tk.Stop()

			// 新的一天
			toChan <- t

			break AlignHour
		}
	}

DayLoop:
	ch = time.Tick(time.Hour * 24)
	for {
		select {
		// 新的一天
		case <-ch:
			toChan <- t
		}
	}
}
