# 虚拟 MFA 桌面端工具

一个基于 [Fyne](https://fyne.io/) 框架构建的虚拟 MFA（TOTP）桌面管理工具，支持跨平台运行（默认针对 Windows 做了字体与显示优化）。
使用 [pquerna/otp](https://github.com/pquerna/otp) 生成动态验证码。

## 功能特性

- **跨平台桌面支持**: 基于 Go 和 Fyne 框架，天然支持跨平台编译。
- **现代化卡片式界面**: 采用 Material Design 风格，蓝色渐变标题栏 + 白色圆角卡片 + 硬件加速阴影，简洁专业。
- **账号管理**:
  - **添加账号**: 支持在应用内通过可视化表单添加新的 MFA 账号（自动过滤非法字符、校验 Base32 字符集与密钥长度）。
  - **删除账号**: 提供直观的删除按钮与二次确认弹窗，安全移除不需要的账号。
- **动态颜色进度条**: 验证码生命周期进度条会根据剩余时间百分比变换颜色（>60% 绿色, >20% 黄色, <20% 红色）。
- **实时刷新**: 实时同步系统时间并更新两步验证码（30秒为一个刷新周期）。
- **一键复制与提示**: 点击显示的验证码数字即可自动复制到系统剪贴板，并弹出 1200ms 后自动关闭的成功提示。
- **搜索过滤**: 支持顶部输入框实时模糊搜索过滤账号列表（不区分大小写）。
- **线程安全**: 使用 `sync.RWMutex` 读写锁保护共享数据，账号增删走统一入口，窗口关闭时无 panic。
- **数据持久化**: 账号数据以 JSON 格式持久化在程序同目录下的 `mfa.json`（文件权限 `0600`），便于随程序一起备份迁移。

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
│       ├── app.go          # 主窗口装配（SetupMainWindow）、appContext、添加/删除账户
│       ├── card.go         # accountCard 组件：单卡片构建、显示状态、setCountdown
│       ├── codegen.go      # codeGen：按 (密钥, 周期) 缓存的 TOTP 生成入口
│       ├── debouncer.go    # 搜索输入防抖合并
│       ├── refresh.go      # liveRefresher：每秒驱动卡片倒计时与验证码刷新
│       └── dialog.go       # 添加账号表单弹窗
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

5. **关于主题**：
   默认使用浅色主题（仅当 `FYNE_THEME` 环境变量未显式设置时生效）。如需切换深色主题，可手动设置 `FYNE_THEME=dark`；注意当前自定义卡片使用了固定浅色背景，深色模式下视觉可能与系统背景不一致。

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

- **自定义主题系统**: 实现了 `MFATheme` 结构，支持动态调整主色调和字体大小，进度条颜色随时间实时变化。利用 Fyne v2.8 新增的 `SizeNameCardRadius`、`SizeNameButtonRadius`、`SizeNameDialogRadius` 统一管理全局圆角。支持通过 `MFA_TEXT_SIZE` 环境变量自定义字体大小。
- **硬件加速阴影**: 卡片使用 Fyne v2.8 新增的 `canvas.Shadow`（`DropShadow` 变体），通过 GPU 着色器渲染，性能优于传统软件阴影。
- **线程安全**: 使用 `sync/atomic` 控制窗口生命周期与强制刷新标记，`sync.RWMutex` 保护 `accounts` / `cards` 切片，账号变更经统一加锁路径写入磁盘。
- **验证码更新**: `accountCard` 持有 `codeText` 缓存；周期切换时通过 `codeGen` 生成并 `widget.Label.SetText` 局部刷新，避免整表 `Refresh`。`codeGen` 按 `(secret, 周期)` 维度缓存 HMAC 计算结果，搜索重建同周期内不再重复生成。
- **密钥标准化**: `NormalizeSecret()` + `ValidateSecret()` 统一处理与校验密钥格式。
- **输入验证**: Base32 字符集与最小长度校验在 model 层完成，表单字段使用 Fyne v2.8 的 `FormItem.Required` 标记必填。
- **错误可观测**: 存储层加载/解析失败时通过 `log.Printf` 输出日志；删除保存失败会弹出错误提示。

## 更新日志

### v2.5 (2026-09-02)
- 🧩 UI 拆分：`SetupMainWindow` 从 446 行降至 ~250 行，抽出 `accountCard` 组件、`liveRefresher` 与 `codeGen` 三个职责单一的单元
- ⚡ TOTP 生成按 `(密钥, 周期)` 缓存；搜索重建同周期内不再重复 HMAC 计算
- ⚡ 搜索框加 150ms 防抖，连续输入抖动期间只触发一次列表重建
- 🧹 `theme` 集中 0.6/0.2 阈值常量，新增 `RemainTextColor` 消除卡片中的 magic number
- 🧹 删除复制分支的 `--- ---` 死代码；修复添加账号后触发的重复渲染
- 🎨 `FYNE_THEME` 不再强制覆盖：未显式设置时给出 light 默认，尊重用户/系统主题偏好
- 🧪 新增 `codegen_test.go`（缓存命中、周期切换、错误密钥）与 `debouncer_test.go`（合并、取消、并发）

### v2.4 (2026-08-04)
- 🔒 修复添加账号时未加锁的并发竞态，账号增删统一经 `appContext` 加锁保存
- ⚡ TOTP 仅在 30 秒周期切换或列表重建时重新生成，进度条每秒更新，减少无效计算与整表刷新
- ✅ 密钥校验下沉到 `model.ValidateSecret`，搜索改为不区分大小写，删除后保留当前搜索条件
- 📁 `go run` 时数据文件回退到工作目录；`mfa.json` 写入权限改为 `0600`
- 🧹 移除 `lancet` 依赖，改用标准库；打包脚本改为自动查找 `fyne`
- ✨ 交互优化：轻量 Toast 替代复制弹窗、空状态提示、倒计时秒数、搜索清除、添加框校验失败不关闭、密钥密文输入、回车提交、重复账号检测、首屏立即出码

### v2.3 (2026-05-27)
- ⬆️ 升级 Fyne 框架至 v2.8.0，支持 Wayland、Accessibility、GPU Shader 等新特性
- 🎨 卡片新增 `canvas.Shadow` 硬件加速阴影（`DropShadow` 变体，GPU 着色器渲染）
- 📐 主题新增 `SizeNameCardRadius`、`SizeNameButtonRadius`、`SizeNameDialogRadius` 统一管理全局圆角
- 📋 表单字段使用 `FormItem.Required` 标记必填，提升交互体验

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

- [fyne.io/fyne/v2](https://github.com/fyne-io/fyne) - 跨平台 UI 框架（v2.8.0）
- [github.com/pquerna/otp](https://github.com/pquerna/otp) - TOTP 验证码生成库
