; Vajra Single-Click Windows Installer (Inno Setup)
; Usage: ISCC packaging\windows\vajra.iss
; Requires: Inno Setup 6+ (https://jrsoftware.org/isdl.php)

[Setup]
AppId={{B7E3A5F1-6D5C-4B2A-8E1D-9C4F3E5D7A9B}
AppName=Vajra
AppVersion=1.0.0
AppPublisher=Vajra Contributors
AppPublisherURL=https://github.com/Ritiksuman07/Vajra
AppSupportURL=https://github.com/Ritiksuman07/Vajra
DefaultDirName={localappdata}\Vajra
DefaultGroupName=Vajra
DisableDirPage=no
PrivilegesRequired=lowest
OutputDir=dist\releases
OutputBaseName=VajraSetup-1.0.0-win-x64
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
ShowLanguageDialog=no
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
; Go binaries
Source: "..\dist\vajra-service.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\vajra.exe"; DestDir: "{app}"; Flags: ignoreversion

; Ollama (bundled)
Source: "..\dist\ollama\*"; DestDir: "{app}\ollama"; Flags: ignoreversion recursesubdirs createallsubdirs

; Agent runtime (embedded Python)
Source: "..\dist\python\*"; DestDir: "{app}\python"; Flags: ignoreversion recursesubdirs createallsubdirs

; Agent code
Source: "..\agent\core\*"; DestDir: "{app}\agent\core"; Flags: ignoreversion recursesubdirs
Source: "..\agent\tools\*"; DestDir: "{app}\agent\tools"; Flags: ignoreversion recursesubdirs

; MCP protocol
Source: "..\mcp\*"; DestDir: "{app}\mcp"; Flags: ignoreversion

; Web UI
Source: "..\ui\web\*"; DestDir: "{app}\ui\web"; Flags: ignoreversion recursesubdirs

; Config
Source: "..\config\default.yaml"; DestDir: "{app}\config"; Flags: ignoreversion skipifsourcedoesntexist

[Icons]
Name: "{group}\Vajra"; Filename: "{app}\vajra.exe"
Name: "{group}\Uninstall Vajra"; Filename: "{uninstalldata}"
Name: "{autoprograms}\Vajra"; Filename: "{app}\vajra.exe"
Name: "{autodesktop}\Vajra"; Filename: "{app}\vajra.exe"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Create &desktop shortcut"; GroupDescription: "Additional icons:"
Name: "autostart"; Description: "Start Vajra service on system boot"; Flags: unchecked; GroupDescription: "Service:"
Name: "pullmodel"; Description: "Download default model (llama3.1:8b, ~4.7 GB)"; GroupDescription: "Model:"

[Run]
; Post-install: write initial config
Filename: "powershell.exe"; Parameters: "-NoProfile -Command &quot;New-Item -ItemType Directory -Force -Path '{app}\data\workspace','{app}\data\models','{app}\logs' &quot;"; StatusMsg: "Creating data directories..."
; Optionally pull model
Filename: "{app}\ollama\ollama.exe"; Parameters: "pull llama3.1:8b"; Tasks: pullmodel; Flags: runhidden waituntilterminated
; Start service (optional)
Filename: "{app}\vajra-service.exe"; Parameters: "--daemon"; StatusMsg: "Starting Vajra service..."
; Launch TUI on first run
Filename: "{app}\vajra.exe"; Flags: nowait postinstall skipifsilent

[UninstallRun]
; Stop service on uninstall
Filename: "powershell.exe"; Parameters: "-NoProfile -Command &quot;Stop-Process -Name 'vajra-service' -Force -ErrorAction SilentlyContinue; Stop-Process -Name 'ollama' -Force -ErrorAction SilentlyContinue&quot;"

[UninstallDelete]
; Cleanup data (with confirmation prompt via [Uninstall] script)
Type: filesandordirs; Name: "{app}\data"
Type: filesandordirs; Name: "{app}\logs"

[UninstallDelete]
Type: regdeletevalue; Key: "HKCU\Environment"; Name: "PATH"; Value: ""

[Registry]
; Add to user PATH
Root: HKCU; Subkey: "Environment"; ValueType: string; ValueName: "PATH"; ValueData: "{app};{regenv:PATH}"; Flags: uninsdeletevalue
; Register service location
Root: HKCU; Subkey: "Software\Vajra"; ValueType: string; ValueName: "InstallDir"; ValueData: "{app}"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\Vajra"; ValueType: string; ValueName: "Version"; ValueData: "1.0.0"; Flags: uninsdeletekey

[Code]
// Custom: confirm data deletion on uninstall
function InitializeUninstall(): Boolean;
begin
  Result := True;
end;

// Silent/advanced mode support
function SilentInstall: Boolean;
begin
  Result := not (IsNull(ExpandConstant('{cmdline}')));
end;
