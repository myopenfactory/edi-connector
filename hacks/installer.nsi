!include LogicLib.nsh
!include nsDialogs.nsh
!include MUI2.nsh
!include StrFunc.nsh
!include shelllink.nsh
!include nsProcess.nsh

Name "EDI-Connector"
BrandingText "myOpenFactory EDI-Connector $%VERSION%"
OutFile "edi-connector_installer.exe"
InstallDir "$PROGRAMFILES64\myOpenFactory\EDI-Connector"
ShowInstDetails show
Icon "logo.ico"

!define MUI_ICON "logo.ico"
!define MUI_UNICON "logo.ico"
!define MUI_FINISHPAGE_RUN ""
!define MUI_FINISHPAGE_RUN_FUNCTION launchApplication
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_TEXT "Installed myOpenFactory EDI-Connector in Version $%VERSION%."
!define MUI_FINISHPAGE_LINK "EDI-Connector Documentation"
!define MUI_FINISHPAGE_LINK_LOCATION "https://docs.myopenfactory.com/edi/protocols/edi-connector/"

!define MUI_PAGE_CUSTOMFUNCTION_PRE welcomePre
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
Page custom pgAuthorizationSettingsPageCreate pgAuthoriziationSettingsPageLeave
Page custom pgOutboundSettingsPageCreate pgOutboundSettingsPageLeave
Page custom pgInboundSettingsPageCreate pgInboundSettingsPageLeave
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

; do not remove this it's intended
${StrRep}

var SettingsDir
Var Dialog
Var linkPath

Var ComponentName

var InitializeConfig
Var UserHWND
Var PasswordHWND
Var OutboundProcessIdHWND
Var OutboundFolderHWND
Var OutboundExtensionHWND
Var OutboundErrorFolderHWND
Var InboundProcessIdHWND
Var InboundFolderHWND

Function .onInit
    SetRegView 64
    StrCpy $ComponentName $(^Name)
    StrCpy $SettingsDir "$COMMONPROGRAMDATA\myOpenFactory\$ComponentName"
FunctionEnd

Function un.onInit
    SetRegView 64
    StrCpy $ComponentName $(^Name)
    StrCpy $SettingsDir "$COMMONPROGRAMDATA\myOpenFactory\$ComponentName"
FunctionEnd

Function welcomePre
    nsProcess::_FindProcess "edi-connector.exe"
    Pop $R0
    ${If} $R0 = 0
        MessageBox MB_OK|MB_ICONEXCLAMATION "$(^Name) currently running! Please stop it and try again." /SD IDOK
        Quit
    ${EndIf}
FunctionEnd

Section "EDI-Connector"
    SectionIn RO

    IfFileExists $INSTDIR +2 0
        CreateDirectory $INSTDIR

    IfFileExists $INSTDIR\edi-connector.exe 0 +2
        Delete $INSTDIR\edi-connector.exe

    SetOutPath $INSTDIR
    File ..\dist\edi-connector_windows_amd64_v1\edi-connector.exe
    File /r ..\THIRD_PARTY
    File logo.ico

    IfFileExists $SettingsDir +2 0
        CreateDirectory $SettingsDir

    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\{97379ac3-c194-45d3-a610-334700f90593}" "DisplayName" "EDI-Connector"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\{97379ac3-c194-45d3-a610-334700f90593}" "DisplayIcon" "$\"$INSTDIR\logo.ico$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\{97379ac3-c194-45d3-a610-334700f90593}" "Publisher" "myOpenFactory Software GmbH"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\{97379ac3-c194-45d3-a610-334700f90593}" "DisplayVersion" "$%VERSION%"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\{97379ac3-c194-45d3-a610-334700f90593}" "UninstallString" "$\"$INSTDIR\uninstaller.exe$\""

    WriteUninstaller $INSTDIR\uninstaller.exe

    nsExec::ExecToLog '"$INSTDIR\edi-connector.exe" service uninstall'
    nsExec::ExecToLog '"$INSTDIR\edi-connector.exe" service install'

    Pop $0
    ${If} $0 != 0
        Abort
    ${EndIf}

    createDirectory "$SMPROGRAMS\myOpenFactory\$(^Name)"

    StrCpy $linkPath "$SMPROGRAMS\myOpenFactory\$ComponentName\Restart $ComponentName.lnk"
	createShortCut "$linkPath" "C:\Windows\System32\cmd.exe" "/k $\"$INSTDIR\edi-connector.exe$\" service restart" "$INSTDIR\logo.ico"
    push $linkPath
    call ShellLinkSetRunAs

    StrCpy $linkPath "$SMPROGRAMS\myOpenFactory\$ComponentName\Start $ComponentName.lnk"
	createShortCut "$linkPath" "C:\Windows\System32\cmd.exe" "/k $\"$INSTDIR\edi-connector.exe$\" service start" "$INSTDIR\logo.ico"
    push $linkPath
    call ShellLinkSetRunAs

    StrCpy $linkPath "$SMPROGRAMS\myOpenFactory\$ComponentName\Stop $ComponentName.lnk"
	createShortCut "$linkPath" "C:\Windows\System32\cmd.exe" "/k $\"$INSTDIR\edi-connector.exe$\" service stop" "$INSTDIR\logo.ico"
    push $linkPath
    call ShellLinkSetRunAs
SectionEnd

Section "Uninstall"
    nsExec::ExecToLog '"$INSTDIR\edi-connector.exe" service uninstall'
    RMDir /R "$SMPROGRAMS\myOpenFactory\$(^Name)"

    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\{97379ac3-c194-45d3-a610-334700f90593}"
    RMDir /R $INSTDIR
SectionEnd

Function pgAuthorizationSettingsPageCreate
    IfFileExists "$SettingsDir\config.yaml" 0 +2
        Abort
    StrCpy $InitializeConfig "true"
    CreateDirectory $SettingsDir

    !insertmacro MUI_HEADER_TEXT "myOpenFactory EDI-Connector Settings" "Provide authorization configuration."

    nsDialogs::Create 1018
    Pop $Dialog

    ${If} $Dialog == error
        Abort
    ${EndIf}

    ${NSD_CreateGroupBox} 5% 5% 90% 40% "Authorization settings"
    Pop $0

    ${NSD_CreateLabel} 15% 16% 20% 10u "Username:"
    Pop $0

    ${NSD_CreateText} 40% 15% 30% 12u ""
    Pop $UserHWND

    ${NSD_CreateLabel} 15% 31% 20% 10u "Password:"
    Pop $0

    ${NSD_CreatePassword} 40% 30% 30% 12u ""
    Pop $PasswordHWND

    nsDialogs::Show
FunctionEnd

Function pgAuthoriziationSettingsPageLeave
    ${NSD_GetText} $UserHWND $0
    ${NSD_GetText} $PasswordHWND $1

    ${If} $0 == ""
    ${OrIf} $1 == ""
        MessageBox MB_RETRYCANCEL|MB_ICONSTOP "All fields are required!" /SD IDCANCEL
        Abort
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml serviceName "EDI-Connector"
    ${If} $0 != error
        DetailPrint "Failed to write serviceName"
        SetErrors
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml username $0
    ${If} $0 != error
        DetailPrint "Failed to write username"
        SetErrors
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml password $1
    ${If} $0 != error
        DetailPrint "Failed to write password"
        SetErrors
    ${EndIf}
FunctionEnd

Function pgOutboundSettingsPageCreate
    ${If} $InitializeConfig != "true"
        Abort
    ${EndIf}

    !insertmacro MUI_HEADER_TEXT "myOpenFactory EDI-Connector Settings" "Provide outbound configuration."

    nsDialogs::Create 1018
    Pop $Dialog

    ${If} $Dialog == error
        Abort
    ${EndIf}

    ${NSD_CreateGroupBox} 5% 5% 90% 80% "Outbound settings"
    Pop $0

    ${NSD_CreateLabel} 15% 16% 20% 10u "Process Id:"
    Pop $0

    ${NSD_CreateText} 40% 15% 30% 12u ""
    Pop $OutboundProcessIdHWND

    ${NSD_CreateLabel} 15% 31% 20% 10u "Folder:"
    Pop $0

    ${NSD_CreateDirRequest} 40% 30% 30% 12u ""
    Pop $OutboundFolderHWND
    ${NSD_CreateBrowseButton} 75% 30% 10% 12u "Browse"
    Pop $0
    nsDialogs::SetUserData $0 $OutboundFolderHWND
    ${NSD_OnClick} $0 onFolderBrowse

    ${NSD_CreateLabel} 15% 46% 20% 10u "Extension:"
    Pop $0

    ${NSD_CreateText} 40% 45% 30% 12u "xml"
    Pop $OutboundExtensionHWND

    
    ${NSD_CreateLabel} 15% 61% 20% 10u "Error Folder:"
    Pop $0

    ${NSD_CreateDirRequest} 40% 60% 30% 12u ""
    Pop $OutboundErrorFolderHWND
    ${NSD_CreateBrowseButton} 75% 60% 10% 12u "Browse"
    Pop $0
    nsDialogs::SetUserData $0 $OutboundErrorFolderHWND
    ${NSD_OnClick} $0 onFolderBrowse

    nsDialogs::Show
FunctionEnd

Function pgOutboundSettingsPageLeave
    ${NSD_GetText} $OutboundProcessIdHWND $0
    ${NSD_GetText} $OutboundFolderHWND $1
    ${NSD_GetText} $OutboundExtensionHWND $2
    ${NSD_GetText} $OutboundErrorFolderHWND $3

    ${If} $0 == ""
    ${OrIf} $1 == ""
    ${OrIf} $2 == ""
    ${OrIf} $3 == ""
        MessageBox MB_RETRYCANCEL|MB_ICONSTOP "All fields are required!" /SD IDCANCEL
        Abort
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml outbounds.0.id $0
    ${If} $0 != error
        DetailPrint "Failed to write outbound process id"
        SetErrors
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml outbounds.0.type FILE
    ${If} $0 != error
        DetailPrint "Failed to write outbound type"
        SetErrors
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml outbounds.0.settings.message.path $1
    ${If} $0 != error
        DetailPrint "Failed to write outbound message path"
        SetErrors
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml outbounds.0.settings.message.extensions.0 $2
    ${If} $0 != error
        DetailPrint "Failed to write outbound message path"
        SetErrors
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml outbounds.0.settings.errorPath $3
    ${If} $0 != error
        DetailPrint "Failed to write error path"
        SetErrors
    ${EndIf}
FunctionEnd

Function pgInboundSettingsPageCreate
    ${If} $InitializeConfig != "true"
        Abort
    ${EndIf}

    !insertmacro MUI_HEADER_TEXT "myOpenFactory EDI-Connector Settings" "Provide inbound configuration."

    nsDialogs::Create 1018
    Pop $Dialog

    ${If} $Dialog == error
        Abort
    ${EndIf}

    ${NSD_CreateGroupBox} 5% 5% 90% 40% "Inbound settings"
    Pop $0

    ${NSD_CreateLabel} 15% 16% 20% 10u "Process Id:"
    Pop $0

    ${NSD_CreateText} 40% 15% 30% 12u ""
    Pop $InboundProcessIdHWND

    ${NSD_CreateLabel} 15% 31% 20% 10u "Folder:"
    Pop $0

    ${NSD_CreateDirRequest} 40% 30% 30% 12u ""
    Pop $InboundFolderHWND
    ${NSD_CreateBrowseButton} 75% 30% 10% 12u "Browse"
    Pop $0
    nsDialogs::SetUserData $0 $InboundFolderHWND
    ${NSD_OnClick} $0 onFolderBrowse

    nsDialogs::Show
FunctionEnd


Function pgInboundSettingsPageLeave
    ${NSD_GetText} $InboundProcessIdHWND $0
    ${NSD_GetText} $InboundFolderHWND $1

    ${If} $0 == ""
    ${OrIf} $1 == ""
        MessageBox MB_RETRYCANCEL|MB_ICONSTOP "All fields are required!" /SD IDCANCEL
        Abort
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml inbounds.0.id $0
    ${If} $0 != error
        DetailPrint "Failed to write inbound process id"
        SetErrors
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml inbounds.0.type FILE
    ${If} $0 != error
        DetailPrint "Failed to write inbound type"
        SetErrors
    ${EndIf}

    nsYaml::write $SettingsDir/config.yaml inbounds.0.settings.path $1
    ${If} $0 != error
        DetailPrint "Failed to write inbound path"
        SetErrors
    ${EndIf}
FunctionEnd

Function onFolderBrowse
    Pop $0
    nsDialogs::GetUserData $0
    Pop $1
    nsDialogs::SelectFolderDialog open ""
    Pop $0
    ${If} $0 != error
        ${NSD_SetText} $1 "$0"
    ${EndIf}
FunctionEnd

Function launchApplication
    Exec "C:\Windows\System32\cmd.exe /k $\"$\"$INSTDIR\edi-connector.exe$\" service start$\""
FunctionEnd

