mkdir -p /tmp/onigirazu-e2e-www && printf hello > /tmp/onigirazu-e2e-www/hello.txt
pkill -f 'http.server 18765' || true
cd /tmp/onigirazu-e2e-www && setsid nohup python3 -m http.server 18765 --bind 127.0.0.1 >/dev/null 2>&1 < /dev/null &
sleep 1
