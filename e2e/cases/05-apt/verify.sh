set -e
dpkg-query -W -f='${Status}' tree | grep -q 'install ok installed'
dpkg-query -W -f='${Status}' jq | grep -q 'install ok installed'
