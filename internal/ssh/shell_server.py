# onigirazu's command server on the host, when it has Python: the protocol of
# shellScript (base64 command lines in, "ONIGIRAZU rc stdout-bytes
# stderr-bytes" plus the bytes out), with one process per command (sh) instead
# of the helpers the POSIX version needs (base64, wc, cat, rm). Output goes
# through files, as there: a background process a command leaves behind keeps
# a removed file open, not a pipe the next answer waits on.
import os, shutil, subprocess, sys, tempfile
from base64 import b64decode

root = os.path.expanduser("~/.onigirazu/tmp")
try:
    # never in the home of another user (sudo may keep HOME): that user's
    # own server could not use the directory afterwards
    if os.stat(os.path.expanduser("~")).st_uid != os.geteuid():
        raise OSError("home of another user")
    os.makedirs(root, mode=0o700, exist_ok=True)
    work = tempfile.mkdtemp(prefix="py.", dir=root)
except OSError:
    work = tempfile.mkdtemp(prefix="onigirazu.")
inp, out = sys.stdin.buffer, sys.stdout.buffer


def read(path):
    try:
        with open(path, "rb") as f:
            return f.read()
    except OSError:
        return b""  # the command removed its own output file


out.write(b"ONIGIRAZU-READY\n")
out.flush()
n = 0
try:
    while True:
        line = inp.readline()
        if not line:
            break
        mode, _, data = line.rstrip(b"\n").partition(b" ")
        n += 1
        try:
            command = b64decode(data, validate=True)
        except Exception:
            out.write(b"ONIGIRAZU 255 0 0\n")
            out.flush()
            continue
        os.makedirs(work, mode=0o700, exist_ok=True)
        script = os.path.join(work, "c")
        o, e = os.path.join(work, "o%d" % n), os.path.join(work, "e%d" % n)
        with open(script, "wb") as f:
            f.write(command)
        with open(o, "wb") as fo, open(e, "wb") as fe:
            rc = subprocess.call(["sh", script], stdin=subprocess.DEVNULL, stdout=fo,
                                 stderr=fo if mode == b"C" else fe)
        if rc < 0:
            rc = 128 - rc  # killed by a signal: as sh reports it
        so, se = read(o), read(e)
        for p in (o, e):
            try:
                os.unlink(p)
            except OSError:
                pass
        out.write(b"ONIGIRAZU %d %d %d\n" % (rc, len(so), len(se)) + so + se)
        out.flush()
finally:
    shutil.rmtree(work, ignore_errors=True)
