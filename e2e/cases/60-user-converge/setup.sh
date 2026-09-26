userdel -r onigirazu-e2e-uc 2>/dev/null || true
userdel -r onigirazu-e2e-uc2 2>/dev/null || true
groupdel onigirazu-e2e-uc 2>/dev/null || true
useradd -m -s /bin/sh -c first -G sys onigirazu-e2e-uc
