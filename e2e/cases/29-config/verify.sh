set -e
python3 -c 'import json; d = json.load(open("/etc/onigirazu-e2e.json")); assert d["server"]["port"] == 8080, d; assert d["server"]["host"] == "0.0.0.0", d'
