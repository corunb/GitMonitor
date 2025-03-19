package main

import (
 "bytes"
 "flag"
 "fmt"
 "os"
 "os/exec"
 "time"
)

type Config struct {
 RepoURL       string        `json:"repo_url"`       // GitHub项目地址
 LocalPath     string        `json:"local_path"`     // 本地同步目录
 CheckInterval time.Duration `json:"check_interval"` // 检测间隔
}

const defaultInterval = 300 * time.Second

func main() {
 // 从命令行参数获取项目地址和本地目录
 repoFlag := flag.String("u", "", "Git repository URL")
 pathFlag := flag.String("p", "", "Local directory path")
 intervalFlag := flag.Duration("t", defaultInterval, "检测间隔 (例如: 10s, 1m)")
 flag.Parse()

 // 如果用户没有输入任意一个参数，则提示并退出
 if *repoFlag == "" || *pathFlag == "" {
    fmt.Println(`用法: 
    gitmonitor -u https://github.com/xxx/xxx.git -p /xxx/xxx [-t 10s/10m/10h]
    -u：指定目标地址。
    -p：指定本地文件夹路径。
    -t：自定义检测频率，10s(10秒)/10m(10分钟)/10h(10小时)，可自定义，默认5分钟。`)
  os.Exit(1)
 }

 cfg := &Config{
  RepoURL:       *repoFlag,
  LocalPath:     *pathFlag,
  CheckInterval: *intervalFlag,
 }

 // 初始化仓库
 if err := initRepo(cfg); err != nil {
  exitWithError(fmt.Sprintf("仓库初始化失败: %v", err))
 }

 fmt.Printf("🛠 开始监控仓库\n  远程地址: %s\n  本地目录: %s\n  检测间隔: %v\n",
  cfg.RepoURL, cfg.LocalPath, cfg.CheckInterval)

 ticker := time.NewTicker(cfg.CheckInterval)
 defer ticker.Stop()

 for range ticker.C {
  checkAndUpdate(cfg)
 }
}

// 初始化仓库
func initRepo(cfg *Config) error {
 // 检查目录是否存在
 if _, err := os.Stat(cfg.LocalPath); os.IsNotExist(err) {
  fmt.Printf("⏳ 本地目录不存在，正在克隆仓库...\n")
  if err := os.MkdirAll(cfg.LocalPath, 0755); err != nil {
   return fmt.Errorf("创建目录失败: %w", err)
  }
  if _, err := runGitCommand(cfg.LocalPath, "clone", cfg.RepoURL, "."); err != nil {
   return fmt.Errorf("克隆仓库失败: %w", err)
  }
  fmt.Println("✅ 仓库克隆成功")
  return nil
 }

 // 验证是否是Git仓库
 if _, err := runGitCommand(cfg.LocalPath, "rev-parse", "--is-inside-work-tree"); err != nil {
  return fmt.Errorf("目录 %s 不是有效的Git仓库", cfg.LocalPath)
 }

 // 验证远程地址是否匹配
 remoteURL, err := runGitCommand(cfg.LocalPath, "remote", "get-url", "origin")
 if err != nil {
  return fmt.Errorf("获取远程地址失败: %w", err)
 }

 if !bytes.Equal(bytes.TrimSpace(remoteURL), []byte(cfg.RepoURL)) {
  return fmt.Errorf("远程地址不匹配\n  配置地址: %s\n  实际地址: %s",
   cfg.RepoURL, remoteURL)
 }

 return nil
}

// 检测并更新仓库（只同步新增和修改的文件，忽略删除）
func checkAndUpdate(cfg *Config) {
 fmt.Printf("\n⏳ 检测时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))

 // 先获取远程更新
 if output, err := runGitCommand(cfg.LocalPath, "fetch", "origin"); err != nil {
  fmt.Printf("⚠️ 获取远程更新失败: %v\n%s", err, output)
  return
 }

 // 获取远程 HEAD 中的所有文件列表
 remoteFilesOutput, err := runGitCommand(cfg.LocalPath, "ls-tree", "-r", "--name-only", "origin/HEAD")
 if err != nil {
  fmt.Printf("⚠️ 获取远程文件列表失败: %v\n", err)
  return
 }
 remoteFiles := bytes.Split(bytes.TrimSpace(remoteFilesOutput), []byte("\n"))

 updated := false
 // 遍历远程文件，检测新增或修改的文件
 for _, file := range remoteFiles {
  fileStr := string(file)
  // 检查该文件在工作区与远程版本是否有差异
  diffOutput, err := runGitCommand(cfg.LocalPath, "diff", "origin/HEAD", "--", fileStr)
  if err != nil {
   fmt.Printf("⚠️ 检查文件 %s 差异失败: %v\n", fileStr, err)
   continue
  }
  if len(diffOutput) > 0 {
   // 如果有差异，则同步更新该文件
   if output, err := runGitCommand(cfg.LocalPath, "checkout", "origin/HEAD", "--", fileStr); err != nil {
    fmt.Printf("❌ 更新文件 %s 失败: %v\n%s", fileStr, err, output)
   } else {
    fmt.Printf("✅ 同步文件: %s\n", fileStr)
    updated = true
   }
  }
 }

 if !updated {
  fmt.Println("✅ 仓库已是最新状态（新增和修改的文件已同步）")
 }
}

// 执行Git命令
func runGitCommand(path string, args ...string) ([]byte, error) {
 baseArgs := []string{"-C", path}
 cmd := exec.Command("git", append(baseArgs, args...)...)

 output, err := cmd.CombinedOutput()
 if err != nil {
  return nil, fmt.Errorf("命令执行失败: %w\n输出: %s", err, output)
 }
 return output, nil
}

// 简化哈希显示
func shortenHash(hash []byte) string {
 if len(hash) >= 7 {
  return string(hash[:7])
 }
 return string(hash)
}

func exitWithError(msg string) {
 fmt.Fprintln(os.Stderr, "错误:", msg)
 os.Exit(1)
}