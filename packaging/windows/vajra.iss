; ─────────────────────────────────────────────────────────────
;  Vajra — Windows installer (Inno Setup 6)
;
;  Build:  ISCC packaging\windows\vajra.iss
;  With a version override:
;          ISCC /DVersion=1.2.0 packaging\windows\vajra.iss
;
;  Expects the compiled binary at dist\vajra.exe
;  (produced by:  go build -o dist\vajra.exe .  )
; ─────────────────────────────────────────────────────────────

#ifndef Version
  #define Version "1.2.0"
#endif

#define AppName "Vajra"
#define AppPublisher "Vajra Contributors"
#define AppURL "https://github.com/Ritiksuman07/Vajra"

[Setup]
AppId={{B7E3A5F1-6D5C-4B2A-8E1D-9C4F3E5D7A9B}
AppName={#AppName}
AppVersion={#Version}
AppVerName={#AppName} {#Version}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}/issues
AppUpdatesURL={#AppURL}/releases
DefaultDirName={localappdata}\Programs\Vajra
DefaultGroupName=Vajra
DisableDirPage=no
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
OutputDir=..\..\dist\releases
OutputBaseFilename=VajraSetup-{#Version}-win-x64
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
ShowLanguageDialog=no
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayName={#AppName}
UninstallDisplayIcon={app}\vajra.exe
LicenseFile=..\..\LICENSE

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut"; GroupDescription: "Additional shortcuts:"
Name: "addtopath";  Description: "Add Vajra to &PATH (run `vajra` from any terminal)"; GroupDescription: "Other:"

[Files]
Source: "..\..\dist\vajra.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\README.md";     DestDir: "{app}\docs"; Flags: ignoreversion
Source: "..\..\LICENSE";       DestDir: "{app}\docs"; Flags: ignoreversion
Source: "..\..\CHANGELOG.md";  DestDir: "{app}\docs"; Flags: ignoreversion

[Icons]
Name: "{group}\Vajra (Terminal UI)"; Filename: "{app}\vajra.exe"
Name: "{group}\Vajra (Web UI)";      Filename: "{app}\vajra.exe"; Parameters: "web"
Name: "{group}\Uninstall Vajra";     Filename: "{uninstallexe}"
Name: "{autoprograms}\Vajra (Terminal UI)"; Filename: "{app}\vajra.exe"
Name: "{autoprograms}\Vajra (Web UI)";      Filename: "{app}\vajra.exe"; Parameters: "web"
Name: "{autodesktop}\Vajra"; Filename: "{app}\vajra.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\vajra.exe"; Parameters: "doctor"; Description: "Run the setup check"; Flags: postinstall nowait skipifsilent shellexec
Filename: "{app}\vajra.exe"; Description: "Launch Vajra"; Flags: postinstall nowait skipifsilent unchecked

[Registry]
Root: HKCU; Subkey: "Software\Vajra"; ValueType: string; ValueName: "InstallDir"; ValueData: "{app}"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\Vajra"; ValueType: string; ValueName: "Version"; ValueData: "{#Version}"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Check: NeedsAddPath('{app}'); Tasks: addtopath

[UninstallDelete]
Type: filesandordirs; Name: "{app}"

[Code]
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKCU, 'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Uppercase(Param) + ';', ';' + Uppercase(OrigPath) + ';') = 0;
end;
