set -e
dpkg-query -W -f='${Status}' cowsay | grep -q 'install ok installed'
