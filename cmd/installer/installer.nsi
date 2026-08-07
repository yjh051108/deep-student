; =============================================================================
; DeepStudent NSIS 安装脚本 (v1.0.0)
; -----------------------------------------------------------------------------
; 目标产物: DeepStudent-Setup-1.0.0.exe (单文件 NSIS 安装包)
; -----------------------------------------------------------------------------
;  打包内容:
;    - build/bin/deepstudent.exe  (Wails 编译产物)
;    - frontend/dist/*            (前端静态资源，递归包含)
;    - LICENSE                     (AGPL-3.0 全文)
;    - README.txt                  (用户面向的安装/卸载说明)
;    - icon.ico                    (多分辨率应用图标)
;  行为约定:
;    - 默认安装到 %ProgramFiles%\DeepStudent (需管理员权限)
;    - 桌面快捷方式 + 开始菜单快捷方式
;    - HKLM 注册表写入 InstallDir / Version / DataDir
;    - 卸载时弹出"是否删除用户数据"复选项，默认勾选"保留"
;    - 安装完成页可勾选"立即启动"
;    - 完整支持中文 / 空格 / Unicode 路径
; =============================================================================

Unicode False
SetCompressor /SOLID lzma
SetDatablockOptimize on

; ----- 头文件 -----
!include "MUI2.nsh"
!include "x64.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"
!include "StrFunc.nsh"
!include "nsDialogs.nsh"

; ----- 应用元信息 -----
!define APP_NAME      "DeepStudent"
!define APP_DISPLAY   "DeepStudent"
!define APP_PUBLISHER "DeepStudent"
!define APP_VERSION   "1.0.0"
!define APP_EXE       "deepstudent.exe"
!define APP_ICON      "icon.ico"
!define INSTALL_KEY   "Software\${APP_PUBLISHER}\${APP_NAME}"
!define UNINST_KEY    "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"
!define DATA_DIR      "DeepStudent"

; ----- 安装器元信息 -----
Name "${APP_DISPLAY} ${APP_VERSION}"
OutFile "..\..\build\installer\DeepStudent-Setup-${APP_VERSION}.exe"
InstallDir "$PROGRAMFILES64\${APP_NAME}"
InstallDirRegKey HKLM "${INSTALL_KEY}" "InstallDir"
RequestExecutionLevel admin
ShowInstDetails show
ShowUninstDetails show
BrandingText "${APP_DISPLAY} ${APP_VERSION}"

; ----- 界面美化 -----
!define MUI_ABORTWARNING
!define MUI_ICON   "${APP_ICON}"
!define MUI_UNICON "${APP_ICON}"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "${NSISDIR}\Contrib\Graphics\Header\nsis.bmp"
!define MUI_WELCOMEPAGE_TITLE "${APP_DISPLAY} 安装向导"
!define MUI_WELCOMEPAGE_TEXT "本安装程序将引导您完成 ${APP_DISPLAY} 在本计算机上的安装。$\r$\n$\r$\n${APP_DISPLAY} 是一个本地优先的 AI 学习工作台，所有数据默认保存在您自己的电脑上。$\r$\n$\r$\n点击下一步继续。"
!define MUI_FINISHPAGE_TITLE "${APP_DISPLAY} 安装完成"
!define MUI_FINISHPAGE_TEXT "${APP_DISPLAY} 已安装到您的计算机。可以选择立即启动。$\r$\n$\r$\n程序目录：$INSTDIR$\r$\n数据目录：$APPDATA\${APP_NAME}"
!define MUI_FINISHPAGE_RUN "$INSTDIR\${APP_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "立即启动 ${APP_DISPLAY}"
!define MUI_FINISHPAGE_SHOWREADME "$INSTDIR\README.txt"
!define MUI_FINISHPAGE_SHOWREADME_TEXT "打开自述文件"
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_LINK "访问项目主页"
!define MUI_FINISHPAGE_LINK_LOCATION "https://github.com/helixnow/deep-student"
!define MUI_LICENSEPAGE_BGCOLOR 0xFFFFFF
; 卸载时弹"确定要卸载吗？"内置确认对话框
!define MUI_UNINSTALLER_CONFIRM

; ----- 安装页顺序 -----
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "license.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; ----- 卸载页顺序 -----
!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
; 卸载前再问"是否保留数据" (BUG-006 修复)
UninstPage custom un.CustomPageCreate un.CustomPageLeave
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_RESERVEFILE_LANGDLL

; =============================================================================
; 安装时主流程
; =============================================================================
Section "主程序（必需）" SEC_MAIN
  SectionIn RO
  SetOutPath "$INSTDIR"

  ; Wails 编译产物
  File "..\..\build\bin\${APP_EXE}"

  ; 前端 dist 全部资源 (递归包含)
  File /r "..\..\frontend\dist\*.*"

  ; 文档 & 图标
  File "..\..\LICENSE"
  File "README.txt"
  File "${APP_ICON}"

  ; ---- 注册表 (HKLM，需要管理员) ----
  WriteRegStr HKLM "${INSTALL_KEY}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${INSTALL_KEY}" "Version"    "${APP_VERSION}"
  WriteRegStr HKLM "${INSTALL_KEY}" "DataDir"    "$APPDATA\${APP_NAME}"
  WriteRegStr HKLM "${INSTALL_KEY}" "Publisher"  "${APP_PUBLISHER}"

  ; ---- 卸载器 ----
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; ---- 控制面板卸载入口 (HKLM) ----
  WriteRegStr   HKLM "${UNINST_KEY}" "DisplayName"     "${APP_DISPLAY}"
  WriteRegStr   HKLM "${UNINST_KEY}" "DisplayVersion"  "${APP_VERSION}"
  WriteRegStr   HKLM "${UNINST_KEY}" "Publisher"       "${APP_PUBLISHER}"
  WriteRegStr   HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr   HKLM "${UNINST_KEY}" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegStr   HKLM "${UNINST_KEY}" "QuietUninstallString" "$INSTDIR\Uninstall.exe /S"
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"
SectionEnd

Section "桌面快捷方式" SEC_DESKTOP
  SetShellVarContext all
  CreateShortcut "$DESKTOP\${APP_DISPLAY}.lnk" \
                 "$INSTDIR\${APP_EXE}" "" \
                 "$INSTDIR\${APP_ICON}" 0
SectionEnd

Section "开始菜单快捷方式" SEC_STARTMENU
  SetShellVarContext all
  CreateDirectory "$SMPROGRAMS\${APP_DISPLAY}"
  CreateShortcut "$SMPROGRAMS\${APP_DISPLAY}\${APP_DISPLAY}.lnk" \
                 "$INSTDIR\${APP_EXE}" "" \
                 "$INSTDIR\${APP_ICON}" 0
  CreateShortcut "$SMPROGRAMS\${APP_DISPLAY}\卸载 ${APP_DISPLAY}.lnk" \
                 "$INSTDIR\Uninstall.exe" "" \
                 "$INSTDIR\${APP_ICON}" 0
SectionEnd

; ----- Section 描述 (组件选择页) -----
LangString DESC_SEC_MAIN     ${LANG_SIMPCHINESE} "${APP_DISPLAY} 主程序（必需）"
LangString DESC_SEC_DESKTOP  ${LANG_SIMPCHINESE} "在桌面创建 ${APP_DISPLAY} 快捷方式"
LangString DESC_SEC_STARTMENU ${LANG_SIMPCHINESE} "在开始菜单创建 ${APP_DISPLAY} 快捷方式"
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_MAIN}      $(DESC_SEC_MAIN)
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_DESKTOP}   $(DESC_SEC_DESKTOP)
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_STARTMENU} $(DESC_SEC_STARTMENU)
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; =============================================================================
; 初始化钩子
; =============================================================================
Function .onInit
  ${IfNot} ${RunningX64}
    MessageBox MB_ICONEXCLAMATION|MB_OK "本程序仅支持 64 位 Windows。"
    Abort
  ${EndIf}
FunctionEnd

; =============================================================================
; 卸载流程
; =============================================================================
Var KeepDataChk
Var KeepDataState

; 默认勾选"保留数据"（保护用户资产）
Function un.onInit
  StrCpy $KeepDataState ${BST_CHECKED}
FunctionEnd

; 卸载选项页 — 提供"是否删除用户数据"复选框
Function un.CustomPageCreate
  nsDialogs::Create 1018
  Pop $0
  ${If} $0 == error
    Abort
  ${EndIf}

  ${NSD_CreateLabel} 0 0 100% 12u "卸载选项"
  Pop $0
  ${NSD_CreateLabel} 0 18u 100% 36u "本卸载程序会移除 ${APP_DISPLAY} 的程序文件。$\r$\n是否同时删除您的学习数据(笔记 / 卡片 / 聊天记录 / 设置 / 缓存)？"
  Pop $0
  ${NSD_CreateCheckbox} 0 60u 100% 12u "保留我的学习数据（推荐）"
  Pop $KeepDataChk
  ${NSD_SetState} $KeepDataChk $KeepDataState
  ${NSD_CreateLabel} 0 80u 100% 36u "不勾选此项将彻底删除以下目录(不可恢复)：$\r$\n$APPDATA\${APP_NAME}  以及  $LOCALAPPDATA\${APP_NAME}"
  Pop $0
  ${NSD_CreateLabel} 0 122u 100% 12u "命令行可用 /KEEPDATA 或 /PURGEDATA 覆盖此选项。"
  Pop $0

  nsDialogs::Show
FunctionEnd

Function un.CustomPageLeave
  ${NSD_GetState} $KeepDataChk $KeepDataState
  ; 命令行覆盖
  ${GetOptions} "$CMDLINE" "/KEEPDATA" $0
  ${IfNot} ${Errors}
    StrCpy $KeepDataState ${BST_CHECKED}
  ${Else}
    ${GetOptions} "$CMDLINE" "/PURGEDATA" $0
    ${IfNot} ${Errors}
      StrCpy $KeepDataState ${BST_UNCHECKED}
    ${EndIf}
  ${EndIf}
FunctionEnd

Section "-Uninstall"
  ; 1. 关掉正在运行的实例
  nsExec::ExecToLog 'taskkill /F /IM "${APP_EXE}" /T'
  Sleep 500

  ; 2. 快捷方式
  SetShellVarContext all
  Delete "$DESKTOP\${APP_DISPLAY}.lnk"
  Delete "$SMPROGRAMS\${APP_DISPLAY}\${APP_DISPLAY}.lnk"
  Delete "$SMPROGRAMS\${APP_DISPLAY}\卸载 ${APP_DISPLAY}.lnk"
  RMDir  "$SMPROGRAMS\${APP_DISPLAY}"

  ; 3. 删除程序文件 (整个 $INSTDIR)
  RMDir /r "$INSTDIR"

  ; 4. BUG-006: 是否保留用户数据
  ;    $KeepDataState == 0  →  强删
  ;    $KeepDataState != 0  →  保留 (默认)
  ${If} $KeepDataState == 0
    SetShellVarContext current
    RMDir /r "$APPDATA\${APP_NAME}"
    RMDir /r "$LOCALAPPDATA\${APP_NAME}"
    ; 旧版本兼容 (历史命名空间)
    RMDir /r "$APPDATA\deepstudent-go"
    RMDir /r "$LOCALAPPDATA\deepstudent-go"
  ${EndIf}

  ; 5. 注册表 (HKLM)
  DeleteRegKey HKLM "${INSTALL_KEY}"
  DeleteRegKey HKLM "${UNINST_KEY}"
SectionEnd
