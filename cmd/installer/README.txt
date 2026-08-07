DeepStudent 1.0.0 — 安装说明
============================

安装位置
--------
默认安装到：

    %LOCALAPPDATA%\Programs\DeepStudent

可执行文件：

    DeepStudent.exe

数据目录（首次启动时自动创建）：

    %APPDATA%\DeepStudent
        ├── deepstudent.db       # SQLite 主库
        ├── blob/                # 内容寻址的二进制块
        ├── keys/                # 加密密钥（AES-256-GCM 主密钥）
        ├── vector/              # 嵌入式向量索引
        ├── cache/               # 运行时缓存
        ├── logs/                # 日志
        └── backups/             # 加密快照

注册表
------
    HKCU\Software\helixnow\DeepStudent
        InstallDir = %LOCALAPPDATA%\Programs\DeepStudent
        Version    = 1.0.0
        DataDir    = %APPDATA%\DeepStudent
        Publisher  = helixnow

    HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\DeepStudent
        标准卸载入口（控制面板 → 程序和功能）

启动方式
--------
- 双击桌面 "DeepStudent" 快捷方式
- 开始菜单 → DeepStudent
- 直接运行 DeepStudent.exe

卸载
----
支持三种方式：
1. 控制面板 → 程序和功能 → DeepStudent → 卸载
2. 开始菜单 → DeepStudent → 卸载 DeepStudent
3. 命令行：`"%LOCALAPPDATA%\Programs\DeepStudent\Uninstall.exe" [/KEEPDATA|/PURGEDATA]`
   - /KEEPDATA  卸载但保留数据目录
   - /PURGEDATA 卸载并彻底清除数据目录（不可恢复）

卸载选项页默认勾选"保留数据"，避免误操作。系统集成商 / 大规模部署场景可在
注册表中通过 QuietUninstallString 静默执行。

常见问题
--------
Q1: 安装包提示"必须使用 64 位 Windows"
A1: 本应用仅构建并分发 64 位（amd64）版本；32 位 Windows 不在支持矩阵中。

Q2: 双击图标无反应 / 闪退
A2: 请确保已安装 Microsoft Edge WebView2 Runtime（Windows 10 1803 以下需手动
    安装；Win10 1903+ / Win11 已自带）。日志写入 DataDir/logs/，可定位具体
    错误。

Q3: 中文 / 空格路径下能否正常启动？
A3: 已在 v1.0.0 全面回归（BUG-005 修复），DataDir 可含中文、空格、emoji。

Q4: 卸载后想再次安装但数据还在？
A4: 默认行为。重新安装后 DataDir 仍在原位置，所有笔记 / 卡片 / 聊天记录
    完整保留。

Q5: 如何彻底迁移到新机器？
A5: 拷贝 %APPDATA%\DeepStudent 到新机器的同名目录，再安装新版本即可。

反馈与报告
----------
- 项目主页：https://github.com/helixnow/deep-student
- Issues  ：https://github.com/helixnow/deep-student/issues
