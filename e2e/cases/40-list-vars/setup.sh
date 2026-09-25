# start without the packages, so the first apply must install both
if dpkg -s toilet >/dev/null 2>&1 || dpkg -s fortune-mod >/dev/null 2>&1; then
  DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 remove -y -qq toilet fortune-mod >/dev/null 2>&1
fi
true
