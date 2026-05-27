# 虚拟 MFA 桌面端工具

一个基于 [Fyne](https://fyne.io/) 框架构建的虚拟 MFA（TOTP）桌面管理工具，支持跨平台运行（默认针对 Windows 做了字体与显示优化）。
使用 [pquerna/otp](https://github.com/pquerna/otp) 生成动态验证码。

## 功能特性

- **跨平台桌面支持**: 基于 Go 和 Fyne 框架，天然支持跨平台编译。
- **现代化卡片式界面**: 采用 Material Design 风格，蓝色渐变标题栏 + 白色圆角卡片，简洁专业。
- **账号管理**:
  - **添加账号**: 支持在应用内通过可视化表单添加新的 MFA 账号（自动过滤非法字符、校验 Base32 字符集与密钥长度）。
  - **删除账号**: 提供直观的删除按钮与二次确认弹窗，安全移除不需要的账号。
- **动态颜色进度条**: 验证码生命周期进度条会根据剩余时间百分比变换颜色（>60% 绿色, >20% 黄色, <20% 红色）。
- **实时刷新**: 实时同步系统时间并更新两步验证码（30秒为一个刷新周期）。
- **一键复制与提示**: 点击显示的验证码数字即可自动复制到系统剪贴板，并弹出 800ms 后自动关闭的成功提示。
- **搜索过滤**: 支持顶部输入框实时模糊搜索过滤账号列表。
- **线程安全**: 使用 `sync.RWMutex` 读写锁保护共享数据，窗口关闭时无 panic，保证应用稳定性。
- **数据持久化**: 账号数据以 JSON 格式持久化在程序同目录下的 `mfa.json`，便于随程序一起备份迁移。

## 项目结构

```text
mfa_reader/
├── main.go                # 程序入口文件，负责初始化 Fyne 应用与拉起主界面
├── internal/              # 内部私有包目录
│   ├── model/             # 核心数据结构 MFAAccount 及密钥标准化方法
│   │   └── account_test.go # model 包单元测试
│   ├── storage/           # JSON 数据的加载与持久化存储（程序同目录 mfa.json）
│   │   └── storage_test.go # storage 包单元测试
│   ├── theme/             # 自定义 Fyne 主题、配色方案与应用图标加载
│   │   └── theme_test.go  # theme 包单元测试
│   └── ui/                # 界面构建、弹窗交互及定时刷新渲染逻辑
├── FyneApp.toml           # Fyne 打包配置文件
├── build_windows.bat      # Windows 一键打包脚本
├── icon.png               # 应用程序图标文件
├── go.mod / go.sum        # 依赖管理文件
└── README.md              # 项目说明文档
```

## 运行与编译

### 1. 运行程序

确保您已经安装了 Go 环境 (>= 1.21)，在项目根目录下执行：

```bash
go run main.go
```

*初次运行后，您可以直接点击界面右上角的"添加"按钮来录入您的 MFA 密钥。*

### 2. 编译为可执行文件

如果只需要一个简单的可执行文件用于日常使用：

```bash
go build -o mfa_reader.exe
```

### 3. 运行测试

```bash
go test ./...
```

## 打包发布 (Windows)

如果需要打包为带有图标且没有命令行黑窗口的正式 Windows GUI 应用程序，请按以下步骤操作：

1. **安装 Fyne 命令行工具**：
   ```bash
   go install fyne.io/fyne/v2/cmd/fyne@latest
   ```

2. **执行打包脚本**：
   ```cmd
   build_windows.bat
   ```
   或手动执行：
   ```powershell
   &"$HOME\go\bin\fyne.exe" package -os windows -icon icon.png
   ```

3. **打包结果**：
   执行完成后，您会在当前目录下看到生成的 `.exe` 安装包程序（例如 `mfa_reader.exe`），双击即可运行。

4. **关于中文字体**：
   程序启动时会自动检测 Windows 系统下的 `simhei.ttf` 或 `msyh.ttf` 以防止中文乱码。如果使用其他操作系统或缺少对应字体，请手动配置环境变量 `FYNE_FONT` 指向有效的 `.ttf` 中文字体文件路径。

## 数据存储

账号数据以 JSON 格式存储在程序可执行文件同目录下：

```
mfa.json
```

数据格式示例：

```json
[
	{
		"accountName": "Google",
		"secret": "JBSWY3DPEHPK3PXP"
	}
]
```

如需备份或迁移，直接复制该文件即可。

## 技术亮点

- **自定义主题系统**: 实现了 `MFATheme` 结构，支持动态调整主色调和字体大小，进度条颜色随时间实时变化。支持通过 `MFA_TEXT_SIZE` 环境变量自定义字体大小。
- **线程安全**: 使用 `sync/atomic` 原子操作控制窗口生命周期，`sync.RWMutex` 读写锁保护 `accounts` 和 `updateItems` 切片的并发访问，goroutine 中通过快照读取避免竞态。
- **数据绑定**: 采用 Fyne 的 `binding.String` 机制驱动验证码文本更新，避免频繁重建 UI 对象。
- **密钥标准化**: 统一的 `NormalizeSecret()` 方法处理密钥格式（去空格、去横线、转大写、去填充符），消除多处重复逻辑。
- **输入验证**: 使用正则表达式验证密钥是否为合法的 Base32 字符集（A-Z, 2-7），防止无效密钥导致生成失败。
- **错误可观测**: 存储层加载/解析失败时通过 `log.Printf` 输出日志，不再静默吞掉错误。

## 更新日志

### v2.2 (2026-05-27)
- 🔒 修复 `accounts` 切片并发访问问题，使用 `sync.RWMutex` 保护读写操作
- ⚡ 优化定时器 goroutine 退出逻辑，避免窗口关闭后 `fyne.Do()` 阻塞
- ✅ 添加 Base32 字符集输入验证，防止无效密钥导致生成失败
- 📏 字体大小支持通过 `MFA_TEXT_SIZE` 环境变量自定义，适配不同 DPI 屏幕
- 🧪 添加单元测试，覆盖 model、storage、theme 核心逻辑

### v2.1 (2026-05-27)
- 🔒 修复 `updateItems` 并发竞态问题，使用 `sync.RWMutex` + 快照机制
- 📁 存储路径改为程序同目录下的 `mfa.json`，便于随程序备份迁移
- 🗑️ 移除脆弱的 CSV 兼容解析，统一使用 JSON 格式
- 🧹 提取 `NormalizeSecret()` 方法，消除密钥清理的重复代码
- 📝 存储层错误改为日志输出，便于排查问题
- 🏗️ 引入 `appContext` 结构体，优化函数参数传递

### v2.0 (2026-04-13)
- ✨ 全新现代化界面设计，Material Design 风格
- 🎨 蓝色渐变标题栏 + 圆角卡片布局
- 🔒 修复并发安全问题，窗口关闭时不再 panic
- ⚡ 优化复制提示响应速度 (1s → 800ms)
- 💄 统一配色方案，使用标准化主题变量

### v1.0 (2026-04-12)
- 初始版本发布
- 支持 MFA 账号添加/删除
- 实现 TOTP 验证码实时生成
- 基础搜索过滤功能

## 依赖库

- [fyne.io/fyne/v2](https://github.com/fyne-io/fyne) - 跨平台 UI 框架
- [github.com/duke-git/lancet/v2](https://github.com/duke-git/lancet) - Go 通用工具函数库（文件检测）
- [github.com/pquerna/otp](https://github.com/pquerna/otp) - TOTP 验证码生成库
