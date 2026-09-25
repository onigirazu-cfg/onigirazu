command -v dnf >/dev/null || command -v yum >/dev/null || exit 0
rpm -q tree >/dev/null 2>&1 && (dnf -y -q remove tree || yum -y -q remove tree) >/dev/null 2>&1
true
