import re

with open("desktop/build/windows/installer/wails_tools.nsh", "r") as f:
    c = f.read()

# 1. ARCH define
c = c.replace("""!ifdef SUPPORTS_AMD64
    !ifdef SUPPORTS_ARM64
        !define ARCH "amd64_arm64"
    !else
        !define ARCH "amd64"
    !endif
!else
    !ifdef SUPPORTS_ARM64
        !define ARCH "arm64"
    !else
        !error "Wails: Undefined ARCH, please provide at least one of ARG_WAILS_AMD64_BINARY or ARG_WAILS_ARM64_BINARY"
    !endif
!endif""", """!ifndef SUPPORTS_AMD64
    !ifndef SUPPORTS_ARM64
        !error "Wails: Undefined ARCH, please provide at least one of ARG_WAILS_AMD64_BINARY or ARG_WAILS_ARM64_BINARY"
    !endif
!endif

!ifdef SUPPORTS_AMD64
    !ifdef SUPPORTS_ARM64
        !define ARCH "amd64_arm64"
    !endif
    !ifndef SUPPORTS_ARM64
        !define ARCH "amd64"
    !endif
!endif

!ifndef SUPPORTS_AMD64
    !ifdef SUPPORTS_ARM64
        !define ARCH "arm64"
    !endif
!endif""")

# 2. Check architecture
c = c.replace("""    ${If} ${AtLeastWin10}
        !ifdef SUPPORTS_AMD64
            ${if} ${IsNativeAMD64}
                Goto ok
            ${EndIf}
        !endif

        !ifdef SUPPORTS_ARM64
            ${if} ${IsNativeARM64}
                Goto ok
            ${EndIf}
        !endif

        IfSilent silentArch notSilentArch
        silentArch:
            SetErrorLevel 65
            Abort
        notSilentArch:
            MessageBox MB_OK "${WAILS_ARCHITECTURE_NOT_SUPPORTED}"
            Quit
    ${else}
        IfSilent silentWin notSilentWin
        silentWin:
            SetErrorLevel 64
            Abort
        notSilentWin:
            MessageBox MB_OK "${WAILS_WIN10_REQUIRED}"
            Quit
    ${EndIf}""", """    ${IfNot} ${AtLeastWin10}
        IfSilent silentWin notSilentWin
        silentWin:
            SetErrorLevel 64
            Abort
        notSilentWin:
            MessageBox MB_OK "${WAILS_WIN10_REQUIRED}"
            Quit
    ${EndIf}

    ${If} ${AtLeastWin10}
        !ifdef SUPPORTS_AMD64
            ${if} ${IsNativeAMD64}
                Goto ok
            ${EndIf}
        !endif

        !ifdef SUPPORTS_ARM64
            ${if} ${IsNativeARM64}
                Goto ok
            ${EndIf}
        !endif

        IfSilent silentArch notSilentArch
        silentArch:
            SetErrorLevel 65
            Abort
        notSilentArch:
            MessageBox MB_OK "${WAILS_ARCHITECTURE_NOT_SUPPORTED}"
            Quit
    ${EndIf}""")

# 3. UninstallString
c = c.replace("""    !ifdef WAILS_INSTALL_SCOPE
      !if "${WAILS_INSTALL_SCOPE}" == "user"
        WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
        WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
        WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
        WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\\${PRODUCT_EXECUTABLE}"
        WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" "$\\"$INSTDIR\\uninstall.exe$\\""
        WriteRegStr HKCU "${UNINST_KEY}" "QuietUninstallString" "$\\"$INSTDIR\\uninstall.exe$\\" /S"
      !else
        WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\\${PRODUCT_EXECUTABLE}"
        WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" "$\\"$INSTDIR\\uninstall.exe$\\""
        WriteRegStr HKLM "${UNINST_KEY}" "QuietUninstallString" "$\\"$INSTDIR\\uninstall.exe$\\" /S"
      !endif
    !else
        WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\\${PRODUCT_EXECUTABLE}"
        WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" "$\\"$INSTDIR\\uninstall.exe$\\""
        WriteRegStr HKLM "${UNINST_KEY}" "QuietUninstallString" "$\\"$INSTDIR\\uninstall.exe$\\" /S"
    !endif""", """    !ifndef WAILS_INSTALL_SCOPE
        WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\\${PRODUCT_EXECUTABLE}"
        WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" "$\\"$INSTDIR\\uninstall.exe$\\""
        WriteRegStr HKLM "${UNINST_KEY}" "QuietUninstallString" "$\\"$INSTDIR\\uninstall.exe$\\" /S"
    !endif
    !ifdef WAILS_INSTALL_SCOPE
      !if "${WAILS_INSTALL_SCOPE}" == "user"
        WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
        WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
        WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
        WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\\${PRODUCT_EXECUTABLE}"
        WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" "$\\"$INSTDIR\\uninstall.exe$\\""
        WriteRegStr HKCU "${UNINST_KEY}" "QuietUninstallString" "$\\"$INSTDIR\\uninstall.exe$\\" /S"
      !endif
      !if "${WAILS_INSTALL_SCOPE}" != "user"
        WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
        WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\\${PRODUCT_EXECUTABLE}"
        WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" "$\\"$INSTDIR\\uninstall.exe$\\""
        WriteRegStr HKLM "${UNINST_KEY}" "QuietUninstallString" "$\\"$INSTDIR\\uninstall.exe$\\" /S"
      !endif
    !endif""")

# 4. EstimatedSize
c = c.replace("""    !ifdef WAILS_INSTALL_SCOPE
      !if "${WAILS_INSTALL_SCOPE}" == "user"
        WriteRegDWORD HKCU "${UNINST_KEY}" "EstimatedSize" "$0"
      !else
        WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"
      !endif
    !else
        WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"
    !endif""", """    !ifndef WAILS_INSTALL_SCOPE
        WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"
    !endif
    !ifdef WAILS_INSTALL_SCOPE
      !if "${WAILS_INSTALL_SCOPE}" == "user"
        WriteRegDWORD HKCU "${UNINST_KEY}" "EstimatedSize" "$0"
      !endif
      !if "${WAILS_INSTALL_SCOPE}" != "user"
        WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"
      !endif
    !endif""")


# 5. DeleteUninstaller
c = c.replace("""    !ifdef WAILS_INSTALL_SCOPE
      !if "${WAILS_INSTALL_SCOPE}" == "user"
        DeleteRegKey HKCU "${UNINST_KEY}"
      !else
        DeleteRegKey HKLM "${UNINST_KEY}"
      !endif
    !else
        DeleteRegKey HKLM "${UNINST_KEY}"
    !endif""", """    !ifndef WAILS_INSTALL_SCOPE
        DeleteRegKey HKLM "${UNINST_KEY}"
    !endif
    !ifdef WAILS_INSTALL_SCOPE
      !if "${WAILS_INSTALL_SCOPE}" == "user"
        DeleteRegKey HKCU "${UNINST_KEY}"
      !endif
      !if "${WAILS_INSTALL_SCOPE}" != "user"
        DeleteRegKey HKLM "${UNINST_KEY}"
      !endif
    !endif""")

# 6. SetShellContext
c = c.replace("""    ${If} ${REQUEST_EXECUTION_LEVEL} == "admin"
        SetShellVarContext all
    ${else}
        SetShellVarContext current
    ${EndIf}""", """    ${If} ${REQUEST_EXECUTION_LEVEL} == "admin"
        SetShellVarContext all
    ${EndIf}
    ${IfNot} ${REQUEST_EXECUTION_LEVEL} == "admin"
        SetShellVarContext current
    ${EndIf}""")

with open("desktop/build/windows/installer/wails_tools.nsh", "w") as f:
    f.write(c)

