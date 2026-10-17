; ─────────────────────────────────────────────────────────────
;  Vajra — Windows installer (NSIS fallback)
;  Requires makensis:  https://nsis.sourceforge.io/Download
;  Build:  makensis packaging\windows\vajra.nsi
; ─────────────────────────────────────────────────────────────

!define APPNAME   "Vajra"
!define VERSION   "1.3.0"
!define PUBLISHER "Vajra Contributors"
!define URL       "https://github.com/Ritiksuman07/Vajra"

Name "${APPNAME}"
OutFile "..\..\dist\releases\VajraSetup-${VERSION}-win-x64-nsis.exe"
InstallDir "$LOCALAPPDATA\Programs\Vajra"
RequestExecutionLevel user

!include MUI2.nsh
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Vajra" SecMain
    SetOutPath "$INSTDIR"
    File "..\..\dist\vajra.exe"
    File "..\..\README.md"
    File "..\..\LICENSE"

    CreateDirectory "$SMPROGRAMS\Vajra"
    CreateShortcut "$SMPROGRAMS\Vajra\Vajra (Terminal UI).lnk" "$INSTDIR\vajra.exe"
    CreateShortcut "$SMPROGRAMS\Vajra\Vajra (Web UI).lnk" "$INSTDIR\vajra.exe" "web"
    CreateShortcut "$SMPROGRAMS\Vajra\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    CreateShortcut "$DESKTOP\Vajra.lnk" "$INSTDIR\vajra.exe"

    WriteUninstaller "$INSTDIR\uninstall.exe"

    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "DisplayName" "${APPNAME}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "DisplayVersion" "${VERSION}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "Publisher" "${PUBLISHER}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra" "URLInfoAbout" "${URL}"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\vajra.exe"
    Delete "$INSTDIR\README.md"
    Delete "$INSTDIR\LICENSE"
    Delete "$INSTDIR\uninstall.exe"
    Delete "$SMPROGRAMS\Vajra\Vajra (Terminal UI).lnk"
    Delete "$SMPROGRAMS\Vajra\Vajra (Web UI).lnk"
    Delete "$SMPROGRAMS\Vajra\Uninstall.lnk"
    Delete "$DESKTOP\Vajra.lnk"
    RMDir "$SMPROGRAMS\Vajra"
    RMDir "$INSTDIR"
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Vajra"
SectionEnd
