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
 DingTalkWebhook string
 DingTalkSecret  string
}

const (
 defaultInterval   = 300 * time.Second
 dingTalkWebhook   = "https://oapi.dingtalk.com/robot/send?access_token=xxxxx"
 dingTalkSecret    = "SECxxxxx"
)

func main() {
 repoFlag := flag.String("u", "", "Git repository URL")
 pathFlag := flag.String("p", "", "Local directory path")
 intervalFlag := flag.Duration("t", defaultInterval, "检测间隔 (例如: 10s, 1m)")
 flag.Parse()

 if *repoFlag == "" || *pathFlag == "" {
  fmt.Println(`用法: 
  gitmonitor -u https://github.com/xxx/xxx.git -p /xxx/xxx [-t 10s/10m/10h]
  -u：指定目标地址。
  -p：指定本地文件夹路径。
  -t：自定义检测频率，默认5分钟。`)
  os.Exit(1)
 }

 cfg := &Config{
  RepoURL:       *repoFlag,
  LocalPath:     *pathFlag,
  CheckInterval: *intervalFlag,
  DingTalkWebhook: dingTalkWebhook,
  DingTalkSecret:  dingTalkSecret,
 }

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

 // 如果目录存在但不是 Git 仓库，则初始化
 if _, err := runGitCommand(cfg.LocalPath, "rev-parse", "--is-inside-work-tree"); err != nil {
  fmt.Println("⚠️ 当前目录不是 Git 仓库，正在初始化为 Git 仓库...")
  if _, err := runGitCommand(cfg.LocalPath, "init"); err != nil {
   return fmt.Errorf("初始化 git 仓库失败: %w", err)
  }
  if _, err := runGitCommand(cfg.LocalPath, "remote", "add", "origin", cfg.RepoURL); err != nil {
   return fmt.Errorf("添加远程仓库失败: %w", err)
  }
  if _, err := runGitCommand(cfg.LocalPath, "fetch", "origin"); err != nil {
   return fmt.Errorf("fetch 失败: %w", err)
  }
  if _, err := runGitCommand(cfg.LocalPath, "reset", "--hard", "origin/HEAD"); err != nil {
   return fmt.Errorf("reset 失败: %w", err)
  }
  fmt.Println("✅ Git 仓库初始化完成并同步")
 }

 return nil
}

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

func generateDingTalkSign(secret, timestamp string) string {
 stringToSign := timestamp + "\n" + secret
 h := hmac.New(sha256.New, []byte(secret))
 h.Write([]byte(stringToSign))
 signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
 return url.QueryEscape(signature)
}

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
