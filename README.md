# 虚拟 MFA 桌面端读取工具

一个基于 [Fyne](https://fyne.io/) 框架构建的虚拟 MFA（TOTP）读取工具，支持 Windows。使用 [duke-git/lancet](https://github.com/duke-git/lancet) 进行底层文件与字符串操作，[pquerna/otp](https://github.com/pquerna/otp) 生成动态验证码。

## 功能特性

- **跨平台桌面支持**: 基于 Go 和 Fyne 框架，天然支持跨平台编译（当前默认优化 Windows 显示）。
- **极简卡片式界面**: 基于 Fyne 现代组件构建，支持亮色模式与清晰的卡片布局。
- **动态颜色进度条**: 验证码生命周期进度条会根据剩余时间百分比变换颜色（>60% 绿色, >20% 黄色, <20% 红色）。
- **实时刷新**: 实时同步时间并更新验证码（30秒一个周期）。
- **一键复制与提示**: 点击显示的验证码数字即可自动复制，并弹出 1 秒自动关闭的复制成功提示。
- **搜索过滤**: 支持顶部输入框实时模糊搜索。
- **配置轻量**: 直接读取同级目录下的 `mfa.txt` 文件即可加载账号信息。
- **线程安全**: 针对 Fyne 2.7+ 的 UI 线程检查进行了深度优化，确保在高频刷新下运行稳定。

## 运行与使用

1. 在项目根目录创建 `mfa.txt` 文件，格式为：
   ```text
   名称,添加时间,Base32格式的Secret
   ```
   *注意：如果缺失添加时间，程序将自动以当前时间补全。*

   示例：
   ```text
   Example_Service:test_user,2026-01-01 12:00:00,JBSWY3DPEHPK3PXP
   Another_App:user@example.com,2026-01-02 14:30:00,HXDMVJECJJWSRB3HWIZR4IFUGFTMXBOZ
   ```

2. 运行或编译：
   ```bash
   go run main.go
   
   # 或者编译为可执行文件
   go build -o mfa_reader.exe main.go
   ```

## 打包发布

如果需要打包为带有图标且没有命令行窗口的正式 Windows 应用程序，请按以下步骤操作：

1. **安装 Fyne 命令行工具**：
   ```bash
   go install fyne.io/fyne/v2/cmd/fyne@latest
   ```

2. **执行打包命令**：
   在 PowerShell 中运行（如果 `fyne.exe` 在您的 Go bin 路径下）：
   ```powershell
   &"$HOME\go\bin\fyne.exe" package -os windows
   ```
   或者直接指定绝对路径（以 `admin` 用户为例）：
   ```powershell
   C:\Users\admin\go\bin\fyne.exe package -os windows
   ```

3. **结果**：
   执行完成后，您会在当前目录下看到生成的 `.exe` 安装包（或独立程序）。

4. **防止中文乱码**：
   程序启动时会自动检测 Windows 系统下的 `simhei.ttf` 或 `msyh.ttf`，如果使用其他系统或缺少字体，请手动配置环境变量 `FYNE_FONT` 指向有效的 `.ttf` 中文字体文件。

## 依赖

- [fyne.io/fyne/v2](https://github.com/fyne-io/fyne) - UI 框架
- [github.com/duke-git/lancet/v2](https://github.com/duke-git/lancet) - Go 通用工具函数库
- [github.com/pquerna/otp](https://github.com/pquerna/otp) - TOTP 生成库