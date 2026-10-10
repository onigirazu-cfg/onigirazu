#!/usr/bin/env python3
"""Writes the goss file of the state bench/site.yml leaves on a host.

The same file checks the hosts of both tools after each run, so a run that
skipped work, or did it differently from ansible-playbook, fails the bench.
Volumes come from the playbook's vars and BENCH_VARS (argv[1], JSON).
"""
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = "/opt/bench"

v = {"bench_files": 200, "bench_users": 20, "bench_lines": 100, "bench_workers": 4}
if len(sys.argv) > 1 and sys.argv[1].strip():
    v.update(json.loads(sys.argv[1]))
files, users, lines, workers = (int(v[k]) for k in ("bench_files", "bench_users", "bench_lines", "bench_workers"))


def sha(name):
    with open(os.path.join(HERE, "files", name), "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()




g = {"package": {}, "file": {}, "user": {}, "group": {}, "service": {}, "command": {},
     "kernel-param": {}, "mount": {}, "http": {}}

for p in ["python3-pymysql", "python3-psycopg2", "python3-docker", "jq", "tree", "unzip", "rsync"]:
    g["package"][p] = {"installed": True}

for d in ["conf", "data", "bin", "archive", "repo", "src", "many"]:
    g["file"][f"{ROOT}/{d}"] = {"exists": True, "filetype": "directory", "mode": "0755", "owner": "root"}
g["file"][f"{ROOT}/data/static.txt"] = {"exists": True, "mode": "0644", "sha256": sha("static.txt")}
g["file"][f"{ROOT}/data/downloaded.txt"] = {"exists": True, "mode": "0644", "sha256": sha("static.txt")}
g["file"][f"{ROOT}/bin/hello.sh"] = {"exists": True, "mode": "0755", "sha256": sha("hello.sh")}
for i in range(files):
    g["file"][f"{ROOT}/many/f{i}.txt"] = {"exists": True, "mode": "0640", "owner": "root",
                                         "contents": [f"/^file {i} on \\S+$/"]}
for i in range(min(20, files)):
    g["file"][f"{ROOT}/many/link{i}"] = {"exists": True, "filetype": "symlink", "linked-to": f"{ROOT}/many/f{i}.txt"}
g["file"][f"{ROOT}/conf/app.conf"] = {
    "exists": True, "mode": "0644",
    "contents": ["[app]", "/^host = \\S+$/", f"workers = {workers}"]
    + [f"/^option_{i} = {(i * 7) % 13}$/" for i in range(lines)]}
g["file"][f"{ROOT}/conf/lines.conf"] = {
    "exists": True, "mode": "0644", "contents": [f"/^key{i} = value{i}$/" for i in range(lines)]}
g["file"][f"{ROOT}/conf/block.conf"] = {
    "exists": True, "mode": "0644",
    "contents": ["# BEGIN ANSIBLE MANAGED BLOCK", "/^managed_a = 1$/", "/^managed_b = 2$/", "# END ANSIBLE MANAGED BLOCK"]}
g["file"][f"{ROOT}/conf/replace.conf"] = {
    "exists": True, "mode": "0644", "contents": ["/^listen = 0\\.0\\.0\\.0$/", "!/127\\.0\\.0\\.1/"]}
g["file"][f"{ROOT}/conf/app.ini"] = {
    "exists": True, "mode": "0644", "contents": ["[main]"] + [f"/^opt{i} = {i}$/" for i in range(20)]}
g["file"][f"{ROOT}/archive/conf.tar.gz"] = {"exists": True, "filetype": "file"}
g["file"][f"{ROOT}/src/conf"] = {"exists": True, "filetype": "directory"}
g["file"][f"{ROOT}/repo/origin.git"] = {"exists": True, "filetype": "directory"}
g["file"][f"{ROOT}/repo/checkout/README"] = {"exists": True, "contents": ["/^one$/"]}

g["group"]["bench"] = {"exists": True}
for i in range(users):
    g["user"][f"bench{i}"] = {"exists": True, "shell": "/bin/bash", "home": f"/home/bench{i}", "groups": ["bench"]}
    g["file"][f"/home/bench{i}"] = {"exists": True, "filetype": "directory", "owner": f"bench{i}"}
if users:
    g["file"]["/home/bench0/.ssh/authorized_keys"] = {
        "exists": True, "owner": "bench0",
        "contents": ["AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"]}
    g["command"]["crontab of bench0"] = {
        "exec": "crontab -l -u bench0", "exit-status": 0,
        "stdout": [f"#Ansible: bench job {i}" for i in range(10)] + [f"/^{i} \\* \\* \\* \\* \\/bin\\/true$/" for i in range(10)]}

g["kernel-param"]["vm.swappiness"] = {"value": "10"}
g["mount"]["/mnt/bench"] = {"exists": True, "filesystem": "tmpfs"}
g["file"]["/etc/fstab"] = {"exists": True, "contents": ["/^tmpfs\\s+\\/mnt\\/bench\\s+tmpfs\\s+size=16m\\s/"]}
g["command"]["timezone"] = {"exec": "timedatectl show -p Timezone --value", "exit-status": 0, "stdout": ["/^UTC$/"]}

for s in ["cron", "ssh", "mariadb", "postgresql"]:
    g["service"][s] = {"running": True}
for s in ["cron", "ssh"]:
    g["service"][s]["enabled"] = True
g["command"]["ufw"] = {"exec": "ufw status", "exit-status": 0,
                       "stdout": ["Status: active", "/^22\\/tcp\\s+ALLOW/", "/^8080\\/tcp\\s+ALLOW/"]}
g["command"]["bench-web container"] = {
    # jq, not --format: goss reads its file as a Go template
    "exec": "docker inspect bench-web | jq -r '.[0] | \"\\(.State.Running) \\(.HostConfig.RestartPolicy.Name) \\(.Config.Image)\"'",
    "exit-status": 0, "stdout": ["true unless-stopped alpine:3.20"]}
g["http"]["http://127.0.0.1:8080/"] = {"status": 200, "body": ["ok"], "timeout": 5000}
g["command"]["mysql database"] = {"exec": "mysql -N -e \"SHOW DATABASES LIKE 'bench'\"", "exit-status": 0, "stdout": ["/^bench$/"]}
g["command"]["mysql user"] = {"exec": "mysql -N -e \"SHOW GRANTS FOR 'bench'@'localhost'\"", "exit-status": 0,
                              "stdout": ["/GRANT ALL PRIVILEGES ON `bench`\\.\\* TO `bench`@`localhost`/"]}
g["command"]["postgresql database"] = {
    "exec": "runuser -u postgres -- psql -tAc \"SELECT 1 FROM pg_database WHERE datname = 'bench'\"", "exit-status": 0, "stdout": ["/^1$/"]}
g["command"]["postgresql role"] = {
    "exec": "runuser -u postgres -- psql -tAc \"SELECT 1 FROM pg_roles WHERE rolname = 'bench'\"", "exit-status": 0, "stdout": ["/^1$/"]}

json.dump(g, sys.stdout, indent=1, sort_keys=True)
print()
