package target

import (
	"bufio"
	"fmt"
	"log"
	"moniter/async"
	"moniter/conf"
	"moniter/db"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProcessCPUStats 存储进程 CPU 统计信息
type ProcessCPUStats struct {
	ID        uint      `gorm:"primaryKey"` // 主键
	IP        string    `gorm:"type:varchar(50);not null"`
	Timestamp time.Time `gorm:"type:datetime"`          // 时间戳
	PID       int       `gorm:"column:pid;not null"`    // 进程 ID
	User      string    `gorm:"column:user;not null"`   // 用户名
	USR       float64   `gorm:"column:usr;not null"`    // 用户空间 CPU 使用率
	System    float64   `gorm:"column:system;not null"` // 系统空间 CPU 使用率
	Guest     float64   `gorm:"column:guest;not null"`  // 虚拟 CPU 使用率
	Wait      float64   `gorm:"column:wait;not null"`
	Total     float64   `gorm:"column:total;not null"`   // 总 CPU 使用率
	Command   string    `gorm:"column:command;not null"` // 命令
}

// CPUMonitor CPU 监控器
type CPUMonitor struct {
	processes []string // 要监控的进程名列表
	interval  int      // 监控间隔（秒）
}

// NewCPUMonitor 创建新的 CPU 监控器
func NewCPUMonitor(processes []string, interval int) *CPUMonitor {
	db.DBConn.AutoMigrate(&ProcessCPUStats{})
	cpuTask := async.CPUTask()
	cpuTask.SetConsumer(BatchCreateCPU)
	cpuTask.Async()
	return &CPUMonitor{
		processes: processes,
		interval:  interval,
	}
}

// StartMonitoring 开始监控进程 CPU 使用情况
func (m *CPUMonitor) StartMonitoring() error {
	// 构建 pidstat 命令
	cmd := exec.Command("pidstat", "-u", strconv.Itoa(m.interval))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %v", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting pidstat: %v", err)
	}

	// 创建 scanner 读取输出
	scanner := bufio.NewScanner(stdout)

	// 跳过头部信息
	for i := 0; i < 3; i++ {
		scanner.Scan()
	}

	//// 打印表头
	//fmt.Printf("%-20s %-8s %-10s %-8s %-8s %-8s %-8s %s\n",
	//	"Timestamp", "PID", "User", "%USR", "%System", "%Guest", "%CPU", "Command")
	//fmt.Println(strings.Repeat("-", 90))

	// 处理每一行输出
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 检查是否是目标进程
		isTargetProcess := false
		for _, process := range m.processes {
			if strings.Contains(line, process) {
				isTargetProcess = true
				break
			}
		}

		if isTargetProcess {
			stats, err := parseCPUStats(line)
			if err != nil {
				log.Printf("Error parsing line '%s': %v", line, err)
				continue
			}
			if err = async.CPUTask().Pub(stats); err != nil {
				if err = db.DBConn.Create(&stats).Error; err != nil {
					log.Printf("CPUMonitor Create ,err : %v", err)
					continue
				}
			}

		}
	}

	return cmd.Wait()
}

func BatchCreateCPU(data []interface{}) {
	cpuData := make([]ProcessCPUStats, len(data))
	for i, i2 := range data {
		cpuData[i] = i2.(ProcessCPUStats)
	}
	if err := db.DBConn.CreateInBatches(cpuData, 1000).Error; err != nil {
		log.Printf("IOMonitor CreateInBatches ,err : %v", err)
	}
}

// parseCPUStats 解析 pidstat 输出行
func parseCPUStats(line string) (ProcessCPUStats, error) {

	fields := strings.Fields(line)
	if len(fields) < 8 {
		return ProcessCPUStats{}, fmt.Errorf("invalid line format: %s", line)
	}

	// 解析 PID
	pid, err := strconv.Atoi(fields[2])
	if err != nil {
		return ProcessCPUStats{}, fmt.Errorf("error parsing PID: %v", err)
	}

	// 解析 CPU 使用率数据
	usr, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		return ProcessCPUStats{}, fmt.Errorf("error parsing USR: %v", err)
	}

	system, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		return ProcessCPUStats{}, fmt.Errorf("error parsing System: %v", err)
	}

	guest, err := strconv.ParseFloat(fields[6], 64)
	if err != nil {
		return ProcessCPUStats{}, fmt.Errorf("error parsing Guest: %v", err)
	}
	wait, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		return ProcessCPUStats{}, fmt.Errorf("error parsing Guest: %v", err)
	}

	total, err := strconv.ParseFloat(fields[8], 64)
	if err != nil {
		return ProcessCPUStats{}, fmt.Errorf("error parsing Total CPU: %v", err)
	}

	return ProcessCPUStats{
		IP:        conf.Sc.IP,
		PID:       pid,
		User:      fields[3],
		USR:       usr,
		System:    system,
		Guest:     guest,
		Wait:      wait,
		Total:     total,
		Command:   fields[len(fields)-1],
		Timestamp: time.Now(),
	}, nil
}

// 使用示例
//func main() {
//	// 定义要监控的进程
//	processes := []string{"mysql", "redis", "clickhouse"}
//
//	// 创建监控器实例（1秒间隔）
//	monitor := NewCPUMonitor(processes, 1)
//
//	// 开始监控
//	fmt.Printf("Starting CPU monitoring for processes: %v\n", processes)
//	if err := monitor.StartMonitoring(); err != nil {
//		log.Fatalf("Error during monitoring: %v", err)
//	}
//}

// 添加信号处理，支持优雅退出
func init() {
	// 可以添加信号处理代码
	// 比如处理 Ctrl+C 等中断信号
}
