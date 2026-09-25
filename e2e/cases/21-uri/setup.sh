mkdir -p /tmp/onigirazu-e2e-www && printf hello > /tmp/onigirazu-e2e-www/hello.txt
pkill -f 'http.server 18765' || true
# the whole background group must let go of the SSH session fds, or ssh waits for it
(cd /tmp/onigirazu-e2e-www && exec setsid python3 -m http.server 18765 --bind 127.0.0.1) >/dev/null 2>&1 </dev/null &
for _ in 1 2 3 4 5; do (echo > /dev/tcp/127.0.0.1/18765) 2>/dev/null && break; sleep 1; done
