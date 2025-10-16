;--------------------------------
; setup.nsi - NSIS installer
;--------------------------------

!ifndef VERSION
  !define VERSION "0.0.0"
!endif

!ifndef YEAR
  !define YEAR "2025"
!endif

!ifndef BUILDDIR
  !define BUILDDIR "build"
!endif

!ifndef FILESIZE
  !define FILESIZE
!endif

!define PRODUCTNAME "ShowAllFiles"
!define COMPANYNAME "Kamaran Layne"
!define ERRGENERIC  "An unexpected error occured. Please try again."
!define COMPANYURL  "https://github.com/kamaranl"

!define APPFILE     "${PRODUCTNAME}.exe"
!define SLUG        "${PRODUCTNAME} v${VERSION}"
!define PRODUCTURL  "${COMPANYURL}/showallfiles"
!define SUPPORTURL  "${PRODUCTURL}/issues"
!define UPDATEURL   "${PRODUCTURL}/releases/latest"
!define UINSTREGKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCTNAME}"

;--------------------------------
; General
  Name "${PRODUCTNAME}"
  OutFile "${BUILDDIR}\setup.exe"
  LicenseData "${BUILDDIR}\LICENSE.txt"
  InstallDir "$LOCALAPPDATA\Programs\${PRODUCTNAME}"
  InstallDirRegKey HKCU "Software\${COMPANYNAME}\${PRODUCTNAME}" "Install_Dir"
  ShowInstDetails show
  RequestExecutionLevel user

;--------------------------------
; Pages
  Page license
  Page directory
  Page instfiles
  UninstPage uninstConfirm
  UninstPage instfiles

;--------------------------------
; Installer section
  Section "Install"
    SectionIn RO
    SetOutPath "$INSTDIR"
    WriteUninstaller "$INSTDIR\Uninstall.exe"
    File "${BUILDDIR}\${APPFILE}"
    File "${BUILDDIR}\Root.crt"
    File "${BUILDDIR}\CA.crt"

    nsExec::ExecToLog 'certutil -user -addstore Root "$INSTDIR\Root.crt"'
    nsExec::ExecToLog 'certutil -user -addstore CA "$INSTDIR\CA.crt"'

    CreateShortCut "$SMSTARTUP\${PRODUCTNAME}.lnk" "$INSTDIR\${APPFILE}"
    CreateShortCut "$SMPROGRAMS\${PRODUCTNAME}.lnk" "$INSTDIR\${APPFILE}"

    WriteRegStr HKCU "Software\${COMPANYNAME}\${PRODUCTNAME}" "Install_Dir" "$INSTDIR"
    WriteRegStr HKCU "${UINSTREGKEY}" "Comments" "Like (MacOS) Finder's AppleShowAllFiles, but for File Explorer on Windows."
    WriteRegStr HKCU "${UINSTREGKEY}" "Contact" "kamaran@layne.dev"
    WriteRegStr HKCU "${UINSTREGKEY}" "DisplayIcon" "$INSTDIR\${APPFILE}"
    WriteRegStr HKCU "${UINSTREGKEY}" "DisplayName" "${PRODUCTNAME}"
    WriteRegStr HKCU "${UINSTREGKEY}" "DisplayVersion" "${VERSION}"
    WriteRegStr HKCU "${UINSTREGKEY}" "HelpLink" "${SUPPORTURL}"
    WriteRegStr HKCU "${UINSTREGKEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKCU "${UINSTREGKEY}" "Publisher" "${COMPANYNAME}"
    WriteRegStr HKCU "${UINSTREGKEY}" "UninstallString" "$INSTDIR\Uninstall.exe"
    WriteRegStr HKCU "${UINSTREGKEY}" "URLUpdateInfo" "${UPDATEURL}"
    WriteRegStr HKCU "${UINSTREGKEY}" "URLInfoAbout" "${PRODUCTURL}"

    WriteRegDWORD HKCU "${UINSTREGKEY}" "EstimatedSize" "${FILESIZE}"
    WriteRegDWORD HKCU "${UINSTREGKEY}" "NoModify" 1
    WriteRegDWORD HKCU "${UINSTREGKEY}" "NoRepair" 1
  SectionEnd

;--------------------------------
; Uninstall section
  Section "Uninstall"
    Delete "$INSTDIR\${APPFILE}"
    Delete "$SMSTARTUP\${PRODUCTNAME}.lnk"
    Delete "$SMPROGRAMS\${PRODUCTNAME}.lnk"
    RMDir /r "$INSTDIR"
    DeleteRegKey HKCU "Software\${COMPANYNAME}\${PRODUCTNAME}"
    DeleteRegKey HKCU "${UINSTREGKEY}"
  SectionEnd

;--------------------------------
; Callbacks
  Function .onInstSuccess
    Delete "$INSTDIR\Root.crt"
    Delete "$INSTDIR\CA.crt"
    Exec "$INSTDIR\${APPFILE}"
  FunctionEnd

  Function .onInstFailed
    MessageBox MB_OK "${ERRGENERIC}"
  FunctionEnd

  Function un.onUninstFailed
    MessageBox MB_OK "${ERRGENERIC}"
  FunctionEnd
