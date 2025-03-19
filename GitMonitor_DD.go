package main

import (
 "bytes"
 "crypto/hmac"
 "crypto/sha256"
 "encoding/base64"
 "encoding/json"
 "flag"
 "fmt"
 "io/ioutil"
 "net/http"
 "net/url"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "time"
)

type Config struct {
 RepoURL       string        `json:"repo_url"`       // GitHub项目地址
 LocalPath     string        `json:"local_path"`     // 本地同步目录
 CheckInterval time.Duration `json:"check_interval"` // 检测间隔
 DingTalkWebhook string        // 钉钉机器人 Webhook
 DingTalkSecret  string        // 钉钉机器人密钥 (可选)
}

const(
 defaultInterval = 300 * time.Second

 // 钉钉 Webhook（如果为空，则不启用通知）
 dingTalkWebhook = "https://oapi.dingtalk.com/robot/send?access_token=xxxxx"

 // 钉钉密钥（为空则不启用加签）
 dingTalkSecret = "SECxxxxx"

) 

func main() {
 // 从命令行参数获取项目地址和本地目录
 repoFlag := flag.String("u", "", "Git repository URL")
 pathFlag := flag.String("p", "", "Local directory path")
 intervalFlag := flag.Duration("t", defaultInterval, "检测间隔 (例如: 10s, 1m)")
 flag.Parse()

 // 使用提示
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
  DingTalkWebhook: dingTalkWebhook,
  DingTalkSecret:  dingTalkSecret,
 }

 // 初始化仓库
 if err := initRepo(cfg); err != nil {
  exitWithError(fmt.Sprintf("仓库初始化失败: %v", err))
 }

 fmt.Printf("🛠 监控仓库\n  远程地址: %s\n  本地目录: %s\n  检测间隔: %v\n",
  cfg.RepoURL, cfg.LocalPath, cfg.CheckInterval)

 ticker := time.NewTicker(cfg.CheckInterval)
 defer ticker.Stop()

 for range ticker.C {
  checkAndUpdate(cfg)
 }
}

// 初始化 Git 仓库
func initRepo(cfg *Config) error {
 if _, err := os.Stat(cfg.LocalPath); os.IsNotExist(err) {
  fmt.Println("⏳ 本地目录不存在，正在克隆...")
  if err := os.MkdirAll(cfg.LocalPath, 0755); err != nil {
   return fmt.Errorf("创建目录失败: %w", err)
  }
  if _, err := runGitCommand(cfg.LocalPath, "clone", cfg.RepoURL, "."); err != nil {
   return fmt.Errorf("克隆失败: %w", err)
  }
  fmt.Println("✅ 仓库克隆成功")
  return nil
 }

 if _, err := runGitCommand(cfg.LocalPath, "rev-parse", "--is-inside-work-tree"); err != nil {
  return fmt.Errorf("目录 %s 不是 Git 仓库", cfg.LocalPath)
 }

 return nil
}

// 仅同步新增/修改的文件，不同步删除文件
func checkAndUpdate(cfg *Config) {
 fmt.Printf("\n⏳ 检测时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))

 if _, err := runGitCommand(cfg.LocalPath, "fetch", "origin"); err != nil {
  fmt.Println("⚠️ 获取远程更新失败")
  return
 }

 remoteFilesOutput, err := runGitCommand(cfg.LocalPath, "ls-tree", "-r", "--name-only", "origin/HEAD")
 if err != nil {
  fmt.Println("⚠️ 获取远程文件列表失败")
  return
 }
 remoteFiles := bytes.Split(bytes.TrimSpace(remoteFilesOutput), []byte("\n"))

 updated := false
 var newFiles []string

 for _, file := range remoteFiles {
  fileStr := string(file)
  diffOutput, err := runGitCommand(cfg.LocalPath, "diff", "origin/HEAD", "--", fileStr)
  if err != nil {
   fmt.Printf("⚠️ 检查 %s 失败\n", fileStr)
   continue
  }
  if len(diffOutput) > 0 {
   localFilePath := filepath.Join(cfg.LocalPath, fileStr)
   if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
    newFiles = append(newFiles, fileStr)
   }

   if _, err := runGitCommand(cfg.LocalPath, "checkout", "origin/HEAD", "--", fileStr); err != nil {
    fmt.Printf("❌ 更新 %s 失败\n", fileStr)
   } else {
    fmt.Printf("✅ 更新 %s\n", fileStr)
    updated = true
   }
  }
 }

 if !updated {
  fmt.Println("✅ 仓库已是最新")
 } else if len(newFiles) > 0 && cfg.DingTalkWebhook != "" {
  message := fmt.Sprintf("新增文件同步：\n%s", formatFileList(newFiles))
  if err := sendDingTalkMessage(cfg.DingTalkWebhook, cfg.DingTalkSecret, message); err != nil {
   fmt.Println("❌ 发送钉钉通知失败")
  } else {
   fmt.Println("✅ 钉钉通知已发送")
  }
 }
}

// 发送钉钉消息（支持加签）
func sendDingTalkMessage(webhook, secret, message string) error {
 var sign string
 var timestamp string

 if secret != "" {
  timestamp = strconv.FormatInt(time.Now().UnixMilli(), 10)
  sign = generateDingTalkSign(secret, timestamp)
  webhook = fmt.Sprintf("%s&timestamp=%s&sign=%s", webhook, timestamp, sign)
 }

 payload := map[string]interface{}{
  "msgtype": "text",
  "text":    map[string]string{"content": message},
 }
 body, _ := json.Marshal(payload)

 resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(body))
 if err != nil {
  return fmt.Errorf("请求失败: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  respBody, _ := ioutil.ReadAll(resp.Body)
  return fmt.Errorf("钉钉返回 %s: %s", resp.Status, string(respBody))
 }

 return nil
}

// 计算钉钉加签
func generateDingTalkSign(secret, timestamp string) string {
 stringToSign := timestamp + "\n" + secret
 h := hmac.New(sha256.New, []byte(secret))
 h.Write([]byte(stringToSign))
 signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
 return url.QueryEscape(signature)
}

// 格式化文件列表
func formatFileList(files []string) string {
 return "- " + string(bytes.Join(stringSliceToByteSlices(files), []byte("\n- ")))
}

func stringSliceToByteSlices(ss []string) [][]byte {
 var bs [][]byte
 for _, s := range ss {
  bs = append(bs, []byte(s))
 }
 return bs
}

// 执行 Git 命令
func runGitCommand(path string, args ...string) ([]byte, error) {
 cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
 output, err := cmd.CombinedOutput()
 if err != nil {
  return nil, fmt.Errorf("命令失败: %w\n输出: %s", err, output)
 }
 return output, nil
}

func exitWithError(msg string) {
 fmt.Fprintln(os.Stderr, "错误:", msg)
 os.Exit(1)
}