# start without the packages, so the first apply must install both
if dpkg -s cowsay >/dev/null 2>&1 || dpkg -s sl >/dev/null 2>&1; then
  DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 remove -y -qq cowsay sl >/dev/null 2>&1
fi
true
