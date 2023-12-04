#!/usr/bin/env bash
# Script inspired on https://getcroc.schollz.com/
set -e
#-------------------------------------------------------------------------------
# DEFAULTS
#-------------------------------------------------------------------------------
PREFIX="${PREFIX:-}"
ANDROID_ROOT="${ANDROID_ROOT:-}"

if [[ -n "${PREFIX}" ]]; then
  INSTALL_PREFIX="${PREFIX}"
else
  INSTALL_PREFIX="/usr/local/bin"
fi

print_banner() {
  cat <<-'EOF'
=============================================================================
  ____                  _
 |  _ \ _   _ _ __ ___ | |__   __ _  __ _
 | |_) | | | | '_ ` _ \| '_ \ / _` |/ _` |
 |  __/| |_| | | | | | | |_) | (_| | (_| |
 |_|    \__,_|_| |_| |_|_.__/ \__,_|\__,_|

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
#   DESCRIPTION:  Attempt to determine architecture of host
#    PARAMETERS:  none
#       RETURNS:  0 = Arch Detected. Also prints detected arch to stdout
#                 1 = Unknown arch
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
#   DESCRIPTION:  Attempts to determine host os using uname
#    PARAMETERS:  none
#       RETURNS:  0 = OS Detected. Also prints detected os to stdout
#                 1 = Unknown OS
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
    basedir=$(dirname "${dest}")
    if [ -w "${basedir}" ]; then
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

    asset_name="pumbaa_${cli_os}_${cli_arch}.tar.gz"
    print_message "== Downloading binary file from github: ${asset_name}" "info"

    uri=$(curl -s https://api.github.com/repos/lmtani/pumbaa/releases/latest | grep "browser_download_url" | cut -d '"' -f 4 | grep "${asset_name}")
    print_message "== Downloading from ${uri}" "info"
    output_filename=$(basename "${uri}")
    tempdir=$(mktemp -d)
    curl -L -o "${tempdir}/${output_filename}" "${uri}" 1>"${tempdir}/curl.out" 2>"${tempdir}/curl.err"
    tar -xf "${tempdir}/${asset_name}" -C "${tempdir}"

    print_message "== Installing in ${prefix}/pumbaa" "info"
    install_file_unix "${tempdir}/pumbaa" "${prefix}/pumbaa"
    print_message "Done!" "ok"
}


main "${INSTALL_PREFIX}"
