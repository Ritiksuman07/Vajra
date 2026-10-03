; Vajra Windows Installer (NSIS) — Fallback
; Requires: makensis (https://nsis.sourceforge.io/Download)
; Usage: makensis packaging\windows\vajra.nsi

Name "Vajra"
OutFile "dist/vajra-1.0.0-windows-x64.exe"
InstallDir "$LOCALAPPDATA\Vajra"

!include MUI2.nsh
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Main"
    SetOutPath "$INSTDIR"
    File "..\dist\vajra-service.exe"
    File "..\dist\vajra.exe"
    CreateDirectory "$INSTDIR\config"
    CreateDirectory "$INSTDIR\data"
    CreateDirectory "$INSTDIR\logs"
    File "..\config\default.yaml"
    CreateDirectory "$SMPROGRAMS\Vajra"
    CreateShortcut "$SMPROGRAMS\Vajra\Vajra.lnk" "$INSTDIR\vajra.exe"
    CreateShortcut "$SMPROGRAMS\Vajra\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    WriteUninstaller "$INSTDIR\uninstall.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "DisplayName" "Vajra"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "DisplayVersion" "1.0.0"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "Publisher" "Vajra Contributors"
    WriteRegStr HKCU "Environment" "VAJRA_DATA_DIR" "$INSTDIR\data"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\vajra-service.exe"
    Delete "$INSTDIR\vajra.exe"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR\config"
    RMDir "$INSTDIR\data"
    RMDir "$INSTDIR\logs"
    RMDir "$INSTDIR"
    RMDir "$SMPROGRAMS\Vajra"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra"
    DeleteRegValue HKCU "Environment" "VAJRA_DATA_DIR"
SectionEnd
