# 虚拟 MFA 桌面端读取工具

一个基于 [Fyne](https://fyne.io/) 框架构建的虚拟 MFA（TOTP）读取工具，支持 Windows。使用 [duke-git/lancet](https://github.com/duke-git/lancet) 进行底层文件与字符串操作，[pquerna/otp](https://github.com/pquerna/otp) 生成动态验证码。

## 功能特性

- **跨平台桌面支持**: 基于 Go 和 Fyne 框架，天然支持跨平台编译（当前默认优化 Windows 显示）。
- **极简界面**: 类似手机端 MFA 软件的简洁展示。
- **实时刷新**: 实时同步时间并更新生命周期进度条（30秒一个周期）。
- **一键复制**: 点击显示的验证码数字即可自动将其复制到剪贴板。
- **搜索过滤**: 支持顶部输入框实时模糊搜索。
- **配置轻量**: 直接读取同级目录下的 `mfa.txt` 文件即可加载账号信息。

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

3. **防止中文乱码**：
   程序启动时会自动检测 Windows 系统下的 `simhei.ttf` 或 `msyh.ttf`，如果使用其他系统或缺少字体，请手动配置环境变量 `FYNE_FONT` 指向有效的 `.ttf` 中文字体文件。

## 依赖

- [fyne.io/fyne/v2](https://github.com/fyne-io/fyne) - UI 框架
- [github.com/duke-git/lancet/v2](https://github.com/duke-git/lancet) - Go 通用工具函数库
- [github.com/pquerna/otp](https://github.com/pquerna/otp) - TOTP 生成库