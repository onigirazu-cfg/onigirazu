rm -rf /tmp/e2e-ms
mkdir -p /tmp/e2e-ms
echo original > /tmp/e2e-ms/existing
userdel e2ems 2>/dev/null || true
