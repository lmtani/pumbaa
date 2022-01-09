#!/usr/bin/env bash
# Script inspired on https://getcroc.schollz.com/
set -e
#-------------------------------------------------------------------------------
# DEFAULTS
#-------------------------------------------------------------------------------
PREFIX="${PREFIX:-}"
ANDROID_ROOT="${ANDROID_ROOT:-}"

if [[ -n "${PREFIX}" ]]; then
  INSTALL_PREFIX="${PREFIX}/bin"
else
  INSTALL_PREFIX="/usr/local/bin"
fi

print_banner() {
  cat <<-'EOF'
=============================================================================
 _____ ______ ________  ____    _ _____ _      _          _____  _     _____
/  __ \| ___ \  _  |  \/  | |  | |  ___| |    | |        /  __ \| |   |_   _|
| /  \/| |_/ / | | | .  . | |  | | |__ | |    | |  ______| /  \/| |     | |
| |    |    /| | | | |\/| | |/\| |  __|| |    | | |______| |    | |     | |
| \__/\| |\ \\ \_/ / |  | \  /\  / |___| |____| |____    | \__/\| |_____| |_
 \____/\_| \_|\___/\_|  |_/\/  \/\____/\_____/\_____/     \____/\_____/\___/

=============================================================================
EOF
}

#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  print_message
#   DESCRIPTION:  Prints a message all fancy like
#    PARAMETERS:  $1 = Message to print
#                 $2 = Severity. info, ok, error, warn
#       RETURNS:  Formatted Message to stdout
#-------------------------------------------------------------------------------
print_message() {
  local message
  local severity
  local red
  local green
  local yellow
  local nc

  message="${1}"
  severity="${2}"
  red='\e[0;31m'
  green='\e[0;32m'
  yellow='\e[1;33m'
  nc='\e[0m'

  case "${severity}" in
    "info" ) echo -e "${nc}${message}${nc}";;
      "ok" ) echo -e "${green}${message}${nc}";;
   "error" ) echo -e "${red}${message}${nc}";;
    "warn" ) echo -e "${yellow}${message}${nc}";;
  esac


}

#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  determine_arch
#   DESCRIPTION:  Attempt to determin architecture of host
#    PARAMETERS:  none
#       RETURNS:  0 = Arch Detected. Also prints detected arch to stdout
#                 1 = Unkown arch
#                 20 = 'uname' not found in path
#-------------------------------------------------------------------------------
determine_arch() {
  local uname_out

  if command -v uname >/dev/null 2>&1; then
    uname_out="$(uname -m)"
    if [[ "${uname_out}" == "" ]]; then
      return 1
    else
      echo "${uname_out}"
      return 0
    fi
  else
    return 20
  fi
}

#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  determine_os
#   DESCRIPTION:  Attempts to determin host os using uname
#    PARAMETERS:  none
#       RETURNS:  0 = OS Detected. Also prints detected os to stdout
#                 1 = Unkown OS
#                 20 = 'uname' not found in path
#-------------------------------------------------------------------------------
determine_os() {
  local uname_out

  if command -v uname >/dev/null 2>&1; then
    uname_out="$(uname)"
    if [[ "${uname_out}" == "" ]]; then
      return 1
    else
      echo "${uname_out}"
      return 0
    fi
  else
    return 20
  fi
}

#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  install_file_linux
#   DESCRIPTION:  Installs a file into a location using 'install'.
#    PARAMETERS:  $1 = file to install
#                 $2 = destination to install the file
#       RETURNS:  0 = File Installed
#                 1 = File not installed
#                 20 = Could not find install command
#-------------------------------------------------------------------------------
install_file_unix() {
  local file
  local prefix
  local rcode

  file="${1}"
  dest="${2}"

  if command -v install >/dev/null 2>&1; then
    if [[ "${EUID}" == "0" ]]; then
      install -m 755 "${file}" "${dest}"
      rcode="${?}"
    else
      if command -v sudo >/dev/null 2>&1; then
        sudo install -m 755 "${file}" "${dest}"
        rcode="${?}"
      else
        rcode="21"
      fi
    fi
  else
    rcode="20"
  fi

  return "${rcode}"
}


main() {
    print_banner

    prefix="${1}"
    cli_base_url="https://github.com/lmtani/cromwell-cli/releases/download"
    version="0.9"
    print_message "== Install prefix set to ${prefix}" "info"

    cli_arch="$(determine_arch)"
    cli_arch_rcode="${?}"
    if [[ "${cli_arch_rcode}" == "0" ]]; then
        print_message "== Architecture detected as ${cli_arch}" "info"
    elif [[ "${cli_arch_rcode}" == "1" ]]; then
        print_message "== Architecture not detected" "error"
        exit 1
    else
        print_message "== 'uname' not found in path. Is it installed?" "error"
        exit 1
    fi

    cli_os="$(determine_os)"
    cli_os_rcode="${?}"
    if [[ "${cli_os_rcode}" == "0" ]]; then
        print_message "== OS detected as ${cli_os}" "info"
    elif [[ "${cli_os_rcode}" == "1" ]]; then
        print_message "== OS not detected" "error"
        exit 1
    else
        print_message "== 'uname' not found in path. Is it installed?" "error"
        exit 1
    fi

    asset_name="cromwell-cli_${version}_${cli_os}_${cli_arch}.tar.gz"
    print_message "== Downloading binary file from github: ${asset_name}" "info"
    uri="${cli_base_url}/v${version}/${asset_name}"
    tempdir=$(mktemp -d)
    wget "${uri}" -P "$tempdir" 1>"${tempdir}/wget.out" 2>"${tempdir}/wget.err"
    tar -xf "${tempdir}/${asset_name}" -C "${tempdir}"

    print_message "== Installing in ${prefix}/cromwell-cli" "info"
    install_file_unix "${tempdir}/cromwell-cli" "${prefix}/cromwell-cli"
    print_message "Done!" "ok"
}


main "${INSTALL_PREFIX}"
