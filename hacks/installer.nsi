!include LogicLib.nsh
!include nsDialogs.nsh
!include MUI2.nsh
!include StrFunc.nsh
!include shelllink.nsh
!include nsProcess.nsh

Name "myOpenFactory Client"
BrandingText "myOpenFactory Client $%VERSION%"
OutFile "myof-client_installer.exe"
InstallDir "$PROGRAMFILES64\myOpenFactory\Client"
ShowInstDetails show
Icon "logo.ico"

!define MUI_ICON "logo.ico"
!define MUI_UNICON "logo.ico"
!define MUI_FINISHPAGE_RUN ""
!define MUI_FINISHPAGE_RUN_FUNCTION launchApplication
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_TEXT "Installed myOpenFactory Client in Version $%VERSION%."
!define MUI_FINISHPAGE_LINK "Client Documentation"
!define MUI_FINISHPAGE_LINK_LOCATION "https://docs.myopenfactory.com/edi/protocols/edi-connector/"

!define MUI_PAGE_CUSTOMFUNCTION_PRE welcomePre
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
Page custom pgSettingsPageCreate pgSettingsPageLeave
Page custom pgServiceSettingsPageCreate pgServiceSettingsPageLeave
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

; do not remove this it's intended
${StrRep}

var SettingsDir
var SettingsDirEscaped
Var Dialog
Var Username
Var Password
Var CertificateFile
Var linkPath

Var ComponentName

Var ServiceUsername
Var ServicePassword

Var UserHWND
Var PasswordHWND
Var CertificateHWND

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
    nsProcess::_FindProcess "myof-client.exe"
    Pop $R0
    ${If} $R0 = 0
        MessageBox MB_OK|MB_ICONEXCLAMATION "$(^Name) currently running! Please stop it and try again." /SD IDOK
        Quit
    ${EndIf}
FunctionEnd

Section "Client"
    SectionIn RO

    IfFileExists $INSTDIR +2 0
        CreateDirectory $INSTDIR

    IfFileExists $INSTDIR\myof-client.exe 0 +2
        Delete $INSTDIR\myof-client.exe

    SetOutPath $INSTDIR
    File ..\dist\myof-client_windows_amd64_v1\myof-client.exe
    File /r ..\THIRD_PARTY
    File logo.ico

    IfFileExists $SettingsDir +2 0
        CreateDirectory $SettingsDir

    IfFileExists $SettingsDir\config.properties skipConfigCreation 0
    ${StrRep} '$SettingsDirEscaped' '$SettingsDir' '\' '\\'
    FileOpen $9 $SettingsDir\config.properties "w"
    FileWrite $9 "username = $Username$\r$\n"
    FileWrite $9 "password = $password$\r$\n"
    FileWrite $9 'clientcert = $SettingsDirEscaped\\certificate.pem$\r$\n'
    FileWrite $9 'log.folder = $SettingsDirEscaped\\logs$\r$\n'
    FileClose $9
    CopyFiles $CertificateFile "$SettingsDir\certificate.pem"
    CreateDirectory $SettingsDir\logs
    skipConfigCreation:


    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory" "DisplayName" "Client"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory" "DisplayIcon" "$\"$INSTDIR\logo.ico$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory" "Publisher" "myOpenFactory Software GmbH"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory" "DisplayVersion" "$%VERSION%"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory" "UninstallString" "$\"$INSTDIR\uninstaller.exe$\""

    WriteUninstaller $INSTDIR\uninstaller.exe
SectionEnd

Section "Service" ServiceSection
    ; remove legacy service
    nsExec::ExecToLog '"$INSTDIR\myof-client.exe" service uninstall --name myof-client'
    Pop $0
    ${If} $0 != 0
        Abort
    ${EndIf}

    ; remove existing
    ReadRegStr $0 HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory" "Service"
    ${If} $0 != ""
        nsExec::ExecToLog '"$INSTDIR\myof-client.exe" service uninstall --name $0'
        Pop $0
        ${If} $0 != 0
            Abort
        ${EndIf}
    ${EndIf}

    ${If} $ServiceUsername == ""
    ${AndIf} $ServicePassword == ""
        nsExec::ExecToLog '"$INSTDIR\myof-client.exe" service install --config "$SettingsDir\config.properties" --name $ComponentName'
    ${Else}
        nsExec::ExecToLog '"$INSTDIR\myof-client.exe" service install --config "$SettingsDir\config.properties" --name $ComponentName --logon $ServiceUsername --password $ServicePassword'
    ${EndIf}

    Pop $0
    ${If} $0 != 0
        Abort
    ${EndIf}

    createDirectory "$SMPROGRAMS\myOpenFactory\$(^Name)"

    StrCpy $linkPath "$SMPROGRAMS\myOpenFactory\$ComponentName\Restart $ComponentName.lnk"
	createShortCut "$linkPath" "C:\Windows\System32\cmd.exe" "/k $\"$INSTDIR\myof-client.exe$\" service restart --name $ComponentName" "$INSTDIR\logo.ico"
    push $linkPath
    call ShellLinkSetRunAs

    StrCpy $linkPath "$SMPROGRAMS\myOpenFactory\$ComponentName\Start $ComponentName.lnk"
	createShortCut "$linkPath" "C:\Windows\System32\cmd.exe" "/k $\"$INSTDIR\myof-client.exe$\" service start --name $ComponentName" "$INSTDIR\logo.ico"
    push $linkPath
    call ShellLinkSetRunAs

    StrCpy $linkPath "$SMPROGRAMS\myOpenFactory\$ComponentName\Stop $ComponentName.lnk"
	createShortCut "$linkPath" "C:\Windows\System32\cmd.exe" "/k $\"$INSTDIR\myof-client.exe$\" service stop --name $ComponentName" "$INSTDIR\logo.ico"
    push $linkPath
    call ShellLinkSetRunAs

    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory" "Service" $ComponentName
SectionEnd

Section "Uninstall"
    ReadRegStr $0 HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory" "Service"
    ${If} $0 != ""
        nsExec::ExecToLog '"$INSTDIR\myof-client.exe" service uninstall --name $0'
        RMDir /R "$SMPROGRAMS\myOpenFactory\$(^Name)"
    ${EndIf}

    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\myOpenFactory"
    RMDir /R $INSTDIR
SectionEnd

Function pgSettingsPageCreate
    ; migrate old settings
    IfFileExists $COMMONPROGRAMDATA\myOpenFactory\Client\config.properties 0 skipMigrateConfig
        DetailPrint "Migrating old configuration."
        CopyFiles $COMMONPROGRAMDATA\myOpenFactory\Client\* $SettingsDir
        ${If} ${Errors}
            DetailPrint "Failed to copy old config!"
        ${Else}
          RMDir /R $COMMONPROGRAMDATA\myOpenFactory\Client
        ${EndIf}
        nsExec::ExecToLog '"$INSTDIR\myof-client.exe" config migrate $COMMONPROGRAMDATA\myOpenFactory\Client\config.properties $SettingsDir\config.properties'
    skipMigrateConfig:

    IfFileExists "$SettingsDir\config.properties" 0 +2
        Abort

    !insertmacro MUI_HEADER_TEXT "myOpenFactory Client Settings" "Provide Client configuration."

    nsDialogs::Create 1018
    Pop $Dialog

    ${If} $Dialog == error
        Abort
    ${EndIf}

    ${NSD_CreateGroupBox} 5% 5% 90% 60% "Credentials"
    Pop $0

    ${NSD_CreateLabel} 15% 16% 20% 10u "Username:"
    Pop $0

    ${NSD_CreateText} 40% 15% 30% 12u ""
    Pop $UserHWND

    ${NSD_CreateLabel} 15% 31% 20% 10u "Password:"
    Pop $0

    ${NSD_CreatePassword} 40% 30% 30% 12u ""
    Pop $PasswordHWND

    ${NSD_CreateLabel} 15% 46% 20% 10u "Certificate:"
    Pop $0

    ${NSD_CreateFileRequest} 40% 45% 30% 12u \
        ""
    Pop $CertificateHWND

    ${NSD_CreateBrowseButton} 75% 45% 10% 12u "Browse"
    Pop $0
    ${NSD_OnClick} $0 onCertificateBrowse

    nsDialogs::Show
FunctionEnd

Function pgSettingsPageLeave
    ${NSD_GetText} $UserHWND $0
    StrCpy $Username "$0"

    ${NSD_GetText} $PasswordHWND $0
    StrCpy $Password "$0"

    ${NSD_GetText} $CertificateHWND $0
    StrCpy $CertificateFile "$0"
FunctionEnd

Function onCertificateBrowse
    ${NSD_GetText} $CertificateHWND $0
    nsDialogs::SelectFileDialog open "" ".pem|*.pem"
    Pop $0
    ${If} $0 != error
        ${NSD_SetText} $CertificateHWND "$0"
    ${EndIf}
FunctionEnd

Function pgServiceSettingsPageCreate
    ${Unless} ${SectionIsSelected} ${ServiceSection}
        Abort
    ${EndUnless}

    !insertmacro MUI_HEADER_TEXT "Service Settings" "Provide service configuration."

    nsDialogs::Create 1018
    Pop $Dialog

    ${If} $Dialog == error
        Abort
    ${EndIf}

    ${NSD_CreateGroupBox} 5% 5% 90% 90% "Service Credentials (Optional)"
    Pop $0

    ${NSD_CreateLabel} 15% 16% 20% 10u "Username:"
    Pop $0

    ${NSD_CreateText} 40% 15% 30% 12u ""
    Pop $UserHWND

    ${NSD_CreateLabel} 15% 26% 20% 10u "Password:"
    Pop $0

    ${NSD_CreatePassword} 40% 25% 30% 12u ""
    Pop $PasswordHWND

    ${NSD_CreateLink} 15% 85% 90% 12u "Information about Usernames"
    Pop $0
    ${NSD_OnClick} $0 onUsernameInfoClick

    nsDialogs::Show
FunctionEnd

Function onUsernameInfoClick

ExecShell "open" "https://docs.myopenfactory.com/edi/protocols/edi-connector/"

FunctionEnd

Function pgServiceSettingsPageLeave
    ${NSD_GetText} $UserHWND $0
    StrCpy $ServiceUsername "$0"

    ${NSD_GetText} $PasswordHWND $0
    StrCpy $ServicePassword "$0"
FunctionEnd

Function launchApplication
    ${If} ${SectionIsSelected} ${ServiceSection}
        Exec "C:\Windows\System32\cmd.exe /k $\"$\"$INSTDIR\myof-client.exe$\" service start --name $ComponentName$\""
    ${Else}
        Exec "C:\Windows\System32\cmd.exe /k $\"$INSTDIR\myof-client.exe$\""
    ${EndIf}
FunctionEnd

