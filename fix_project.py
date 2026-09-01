with open("desktop/build/windows/installer/project.nsi", "r") as f:
    content = f.read()

old_block = """!ifdef WAILS_INSTALL_SCOPE
  !if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\\Programs\\${INFO_PRODUCTNAME}"
  !else
    InstallDir "$PROGRAMFILES64\\${INFO_COMPANYNAME}\\${INFO_PRODUCTNAME}"
  !endif
!else
  InstallDir "$PROGRAMFILES64\\${INFO_COMPANYNAME}\\${INFO_PRODUCTNAME}"
!endif # Default installing folder ($PROGRAMFILES is Program Files folder)."""

new_block = """InstallDir "$PROGRAMFILES64\\${INFO_COMPANYNAME}\\${INFO_PRODUCTNAME}" # Default installing folder ($PROGRAMFILES is Program Files folder).
!ifdef WAILS_INSTALL_SCOPE
  !if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\\Programs\\${INFO_PRODUCTNAME}"
  !endif
!endif"""

content = content.replace(old_block, new_block)

with open("desktop/build/windows/installer/project.nsi", "w") as f:
    f.write(content)
