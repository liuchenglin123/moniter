package main

import (
	"fmt"
	"moniter/async"
	"moniter/conf"
	"moniter/db"
	"moniter/target"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ProcessMetrics 结构体定义
type ProcessMetrics struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	IP          string    `gorm:"type:varchar(50);not null"`
	ProcessName string    `gorm:"type:varchar(50);not null"`
	CPUUsage    float64   `gorm:"type:float;not null"`
	MemoryUsage float64   `gorm:"type:float;not null"`
	Timestamp   time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

func (ProcessMetrics) TableName() string {
	return "process_metrics"
}

func init() {
	conf.Init()
	db.Init()

}

func main() {
	// 监控的进程名列表
	processNames := conf.Sc.ProcessNames

	// 创建监控器实例（1秒间隔）
	cpuMonitor := target.NewCPUMonitor(processNames, conf.Sc.IntervalTime)
	memoryMonitor := target.NewMemoryMonitor(processNames, conf.Sc.IntervalTime)
	ioMonitor := target.NewIOMonitor(processNames, conf.Sc.IntervalTime)
	go func() {
		fmt.Println("开始监控进程")
		cpuMonitor.StartMonitoring()
	}()

	go func() {
		fmt.Println("开始监控内存")
		memoryMonitor.StartMonitoring()
	}()
	go func() {
		fmt.Println("开始监控IO")
		ioMonitor.StartMonitoring()
	}()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	defer func() {
		signal.Stop(ch)
		close(ch)
	}()
	<-ch

	// 将留在channel中的数据消费完
	async.ShutDown()

	//// 每秒采集一次数据
	//ticker := time.NewTicker(3 * time.Second)
	////ticker := time.NewTicker(time.Duration(Sc.IntervalTime) * time.Second)
	//defer ticker.Stop()
	//
	//for range ticker.C {
	//	metricsArr := make([]*ProcessMetrics, 0)
	//	for _, procName := range processNames {
	//		metrics, err := collectProcessMetrics(procName)
	//		if err != nil {
	//			log.Printf("采集 %s 指标失败: %v", procName, err)
	//			continue
	//		}
	//		fmt.Println(metrics)
	//		fmt.Println(fmt.Sprintf("采集 %s 指标成功: CPU=%.2f%%, 内存=%.2fMB", procName, metrics.CPUUsage, metrics.MemoryUsage))
	//		metricsArr = append(metricsArr, metrics)
	//	}
	//	if len(metricsArr) == 0 {
	//		continue
	//	}
	//	//err := saveMetrics(metricsArr)
	//	//if err != nil {
	//	//	log.Printf("保存指标失败: %v", err)
	//	//}
	//}
}

//func collectProcessMetrics(processName string) (*ProcessMetrics, error) {
//	processes, err := process.Processes()
//	if err != nil {
//		return nil, err
//	}
//
//	for _, p := range processes {
//		name, err := p.Name()
//		if err != nil {
//			continue
//		}
//
//		if name == processName {
//			cpuPercent, err := p.CPUPercent()
//			if err != nil {
//				return nil, err
//			}
//
//			memInfo, err := p.MemoryInfo()
//			if err != nil {
//				return nil, err
//			}
//
//			// 转换为MB
//			memoryUsageMB := float64(memInfo.RSS) / 1024 / 1024
//
//			return &ProcessMetrics{
//				ProcessName: processName,
//				IP:          Sc.IP,
//				CPUUsage:    cpuPercent,
//				MemoryUsage: memoryUsageMB,
//			}, nil
//		}
//	}
//
//	return nil, fmt.Errorf("进程 %s 未找到", processName)
//}
//
//func collectProcessMetrics1(processName string, interval time.Duration) (*ProcessMetrics, error) {
//	processes, err := process.Processes()
//	if err != nil {
//		return nil, err
//	}
//
//	for _, p := range processes {
//		name, err := p.Name()
//		if err != nil {
//			continue
//		}
//
//		if name == processName {
//			// 获取进程的CPU时间
//			times, err := p.Times()
//			if err != nil {
//				return nil, err
//			}
//
//			currentCPUTime := times.User + times.System
//			currentTime := time.Now()
//
//			// 获取或创建进程的CPU跟踪器
//			tracker, exists := processTrackers[processName]
//			if !exists {
//				tracker = &ProcessCPUTracker{
//					lastCPUTime: currentCPUTime,
//					lastTime:    currentTime,
//				}
//				processTrackers[processName] = tracker
//				// 第一次采集时返回0，因为需要两个采样点才能计算CPU使用率
//				return &ProcessMetrics{
//					ProcessName: processName,
//					CPUUsage:    0,
//					MemoryUsage: 0,
//				}, nil
//			}
//
//			// 计算CPU使用率
//			// CPU使用率 = (CPU时间差 / 总时间差) * 100 * CPU核心数
//			cpuTime := currentCPUTime - tracker.lastCPUTime
//			elapsed := currentTime.Sub(tracker.lastTime).Seconds()
//
//			cpuPercent := (cpuTime / elapsed) * 100 * float64(cpuNum)
//
//			// 更新跟踪器
//			tracker.lastCPUTime = currentCPUTime
//			tracker.lastTime = currentTime
//
//			// 获取内存使用情况
//			memInfo, err := p.MemoryInfo()
//			if err != nil {
//				return nil, err
//			}
//			memoryUsageMB := float64(memInfo.RSS) / 1024 / 1024
//
//			return &ProcessMetrics{
//				ProcessName: processName,
//				CPUUsage:    cpuPercent,
//				MemoryUsage: memoryUsageMB,
//			}, nil
//		}
//	}
//
//	return nil, fmt.Errorf("进程 %s 未找到", processName)
//}
//
//func saveMetrics(metrics []*ProcessMetrics) error {
//	err := DBConn.CreateInBatches(metrics, 10).Error
//	return err
//}
