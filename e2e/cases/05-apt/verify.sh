set -e
dpkg-query -W -f='${Status}' figlet | grep -q 'install ok installed'
dpkg-query -W -f='${Status}' sl | grep -q 'install ok installed'
