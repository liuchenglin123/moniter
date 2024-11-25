package target

import (
	"bufio"
	"fmt"
	"log"
	"mon/async"
	"mon/conf"
	"mon/db"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProcessMemStats 存储进程内存统计信息
type ProcessMemStats struct {
	ID          uint      `gorm:"primaryKey"` // 主键
	IP          string    `gorm:"type:varchar(50);not null"`
	Timestamp   time.Time `gorm:"type:datetime"`                // 时间戳
	PID         int       `gorm:"column:pid;not null"`          // 进程 ID
	User        string    `gorm:"column:user;not null"`         // 用户名
	MinorFaults float64   `gorm:"column:minor_faults;not null"` // 次缺页错误
	MajorFaults float64   `gorm:"column:major_faults;not null"` // 主缺页错误
	VSZ         float64   `gorm:"column:vsz;not null"`          // 虚拟内存大小 (KB)
	RSS         float64   `gorm:"column:rss;not null"`          // 物理内存大小 (KB)
	Command     string    `gorm:"column:command;not null"`      // 命令
}

// MemoryMonitor 内存监控器
type MemoryMonitor struct {
	processes []string  // 要监控的进程名列表
	interval  int       // 监控间隔（秒）
	stopChan  chan bool // 用于停止监控的通道
}

// NewMemoryMonitor 创建新的内存监控器
func NewMemoryMonitor(processes []string, interval int) *MemoryMonitor {
	db.DBConn.AutoMigrate(&ProcessMemStats{})
	memoryTask := async.MemoryTask()
	memoryTask.SetConsumer(BatchCreateMemory)
	memoryTask.Async()
	return &MemoryMonitor{
		processes: processes,
		interval:  interval,
		stopChan:  make(chan bool),
	}
}

// StartMonitoring 开始监控进程内存使用情况
func (m *MemoryMonitor) StartMonitoring() error {
	// 构建命令和管道
	cmd := exec.Command("pidstat", "-r", strconv.Itoa(m.interval))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %v", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting pidstat: %v", err)
	}

	// 创建scanner读取输出
	scanner := bufio.NewScanner(stdout)

	// 跳过头部信息
	for i := 0; i < 3; i++ {
		scanner.Scan()
	}

	//// 处理每一行输出
	//fmt.Printf("%-20s %-8s %-10s %-10s %-10s %-10s %-10s %s\n",
	//	"Timestamp", "PID", "User", "MinFlt/s", "MajFlt/s", "VSZ(KB)", "RSS(KB)", "Command")
	//fmt.Println(strings.Repeat("-", 100))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 检查是否是我们要监控的进程
		isTargetProcess := false
		for _, process := range m.processes {
			if strings.Contains(line, process) {
				isTargetProcess = true
				break
			}
		}

		if isTargetProcess {
			stats, err := parseMemoryStats(line)
			if err != nil {
				log.Printf("Error parsing line '%s': %v", line, err)
				continue
			}
			if err = async.MemoryTask().Pub(stats); err != nil {
				if err = db.DBConn.Create(&stats).Error; err != nil {
					log.Printf("MemoryMonitor Create ,err : %v", err)
					continue
				}
			}

		}
	}

	return cmd.Wait()
}

func BatchCreateMemory(data []interface{}) {
	memoryData := make([]ProcessMemStats, len(data))
	for i, i2 := range data {
		memoryData[i] = i2.(ProcessMemStats)
	}
	if err := db.DBConn.CreateInBatches(memoryData, 1000).Error; err != nil {
		log.Printf("IOMonitor CreateInBatches ,err : %v", err)
	}
}

// parseMemoryStats 解析pidstat输出行
func parseMemoryStats(line string) (ProcessMemStats, error) {
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return ProcessMemStats{}, fmt.Errorf("invalid line format: %s", line)
	}

	// 解析PID
	pid, err := strconv.Atoi(fields[2])
	if err != nil {
		return ProcessMemStats{}, fmt.Errorf("error parsing PID: %v", err)
	}

	// 解析其他数值字段
	minFaults, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		return ProcessMemStats{}, fmt.Errorf("error parsing minor faults: %v", err)
	}

	majFaults, err := strconv.ParseFloat(fields[6], 64)
	if err != nil {
		return ProcessMemStats{}, fmt.Errorf("error parsing major faults: %v", err)
	}

	vsz, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		return ProcessMemStats{}, fmt.Errorf("error parsing VSZ: %v", err)
	}

	rss, err := strconv.ParseFloat(fields[8], 64)
	if err != nil {
		return ProcessMemStats{}, fmt.Errorf("error parsing RSS: %v", err)
	}

	return ProcessMemStats{
		IP:          conf.Sc.IP,
		PID:         pid,
		User:        fields[3],
		MinorFaults: minFaults,
		MajorFaults: majFaults,
		VSZ:         vsz,
		RSS:         rss,
		Command:     fields[len(fields)-1],
		Timestamp:   time.Now(),
	}, nil
}

//func main() {
//	// 定义要监控的进程
//	processes := []string{"mysql", "redis", "clickhouse"}
//
//	// 创建监控器实例，设置1秒的监控间隔
//	monitor := NewMemoryMonitor(processes, 1)
//
//	// 开始监控
//	fmt.Printf("Starting memory monitoring for processes: %v\n", processes)
//	if err := monitor.StartMonitoring(); err != nil {
//		log.Fatalf("Error during monitoring: %v", err)
//	}
//}
