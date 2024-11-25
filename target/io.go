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

// ProcessIOStats 存储进程 IO 统计信息
type ProcessIOStats struct {
	ID        uint      `gorm:"primaryKey"`    // 主键
	Timestamp time.Time `gorm:"type:datetime"` // 时间戳
	IP        string    `gorm:"type:varchar(50);not null"`
	PID       int       `gorm:"column:pid;not null"`        // 进程 ID
	User      string    `gorm:"column:user;not null"`       // 用户名
	ReadKBPS  float64   `gorm:"column:read_kbps;not null"`  // 每秒读取 KB
	WriteKBPS float64   `gorm:"column:write_kbps;not null"` // 每秒写入 KB
	KBCCWR    float64   `gorm:"column:kbccwr;not null"`     // 每秒读取操作次数
	IODelay   float64   `gorm:"column:io_delay;not null"`   // 每秒写入操作次数
	Command   string    `gorm:"column:command;not null"`    // 命令
}

// IOMonitor IO 监控器
type IOMonitor struct {
	processes []string // 要监控的进程名列表
	interval  int      // 监控间隔（秒）
}

// NewIOMonitor 创建新的 IO 监控器
func NewIOMonitor(processes []string, interval int) *IOMonitor {
	db.DBConn.AutoMigrate(&ProcessIOStats{})
	ioTask := async.IOTask()
	ioTask.SetConsumer(BatchCreateIO)
	ioTask.Async()
	return &IOMonitor{
		processes: processes,
		interval:  interval,
	}
}

// StartMonitoring 开始监控进程 IO 使用情况
func (m *IOMonitor) StartMonitoring() error {
	// 构建 pidstat 命令
	cmd := exec.Command("pidstat", "-d", strconv.Itoa(m.interval))
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
	//fmt.Printf("%-20s %-8s %-10s %-10s %-10s %-10s %-10s %s\n",
	//	"Timestamp", "PID", "User", "kB_rd/s", "kB_wr/s", "kB_ccwr/s", "iodelay", "Command")
	//fmt.Println(strings.Repeat("-", 100))

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
			stats, err := parseIOStats(line)
			if err != nil {
				log.Printf("Error parsing line '%s': %v", line, err)
				continue
			}

			if err = async.IOTask().Pub(stats); err != nil {
				if err = db.DBConn.Create(&stats).Error; err != nil {
					log.Printf("IOMonitor Create ,err : %v", err)
					continue
				}
			}
		}
	}

	return cmd.Wait()
}

func BatchCreateIO(data []interface{}) {
	ioData := make([]ProcessIOStats, len(data))
	for i, i2 := range data {
		ioData[i] = i2.(ProcessIOStats)
	}
	if err := db.DBConn.CreateInBatches(ioData, 1000).Error; err != nil {
		log.Printf("IOMonitor CreateInBatches ,err : %v", err)
	}
}

// parseIOStats 解析 pidstat 输出行
func parseIOStats(line string) (ProcessIOStats, error) {
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return ProcessIOStats{}, fmt.Errorf("invalid line format: %s", line)
	}

	// 解析 PID
	pid, err := strconv.Atoi(fields[2])
	if err != nil {
		return ProcessIOStats{}, fmt.Errorf("error parsing PID: %v", err)
	}

	// 解析 IO 数据
	readKB, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		return ProcessIOStats{}, fmt.Errorf("error parsing kB_rd/s: %v", err)
	}

	writeKB, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		return ProcessIOStats{}, fmt.Errorf("error parsing kB_wr/s: %v", err)
	}

	kB_ccwr, err := strconv.ParseFloat(fields[6], 64)
	if err != nil {
		return ProcessIOStats{}, fmt.Errorf("error parsing IO_rd/s: %v", err)
	}

	ioDelay, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		return ProcessIOStats{}, fmt.Errorf("error parsing IO_wr/s: %v,fields:%v", err, fields[7])
	}

	return ProcessIOStats{
		IP:        conf.Sc.IP,
		PID:       pid,
		User:      fields[3],
		ReadKBPS:  readKB,
		WriteKBPS: writeKB,
		KBCCWR:    kB_ccwr,
		IODelay:   ioDelay,
		Command:   fields[len(fields)-1],
		Timestamp: time.Now(),
	}, nil
}

//func main() {
//	// 设置日志格式
//	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
//
//	// 定义要监控的进程
//	processes := []string{"mysql", "redis", "clickhouse"}
//
//	// 创建监控器实例（1秒间隔）
//	monitor := NewIOMonitor(processes, 1)
//
//	// 开始监控
//	fmt.Printf("Starting IO monitoring for processes: %v\n", processes)
//	if err := monitor.StartMonitoring(); err != nil {
//		log.Fatalf("Error during monitoring: %v", err)
//	}
//}

// 添加数据库存储功能
type DBStore struct {
	// 可以添加数据库连接配置
}

func (db *DBStore) StoreIOStats(stats ProcessIOStats) error {
	// 实现数据库存储逻辑
	return nil
}

// 添加告警功能
type AlertConfig struct {
	ReadKBPSThreshold  float64
	WriteKBPSThreshold float64
	IOReadPSThreshold  float64
	IOWritePSThreshold float64
}

func checkAlerts(stats ProcessIOStats, config AlertConfig) {
	// 实现告警检查逻辑
}
