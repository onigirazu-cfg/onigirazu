# onigirazu's command server on the host, when it has Python: the protocol of
# shellScript (base64 command lines in, "ONIGIRAZU rc stdout-bytes
# stderr-bytes" plus the bytes out), with one process per command (sh) instead
# of the helpers the POSIX version needs (base64, wc, cat, rm). Output goes
# through files, as there: a background process a command leaves behind keeps
# a removed file open, not a pipe the next answer waits on.
#
# "P" requests describe a file without a process (the capture before a file
# task): payload "limit path", answer as the shell probe of
# internal/modules/capture.go prints it.
import grp, hashlib, os, pwd, shutil, stat, subprocess, sys, tempfile
from base64 import b64decode, b64encode

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


def probe(request):
    limit, _, path = request.partition(b" ")
    try:
        st = os.lstat(path)
    except FileNotFoundError:
        return b"absent\n"
    if stat.S_ISLNK(st.st_mode):
        kind = b"link"
    elif stat.S_ISDIR(st.st_mode):
        kind = b"directory"
    elif stat.S_ISREG(st.st_mode):
        kind = b"file"
    else:
        kind = b"other"
    try:
        owner = pwd.getpwuid(st.st_uid).pw_name
    except KeyError:
        owner = "UNKNOWN"
    try:
        group = grp.getgrgid(st.st_gid).gr_name
    except KeyError:
        group = "UNKNOWN"
    s = b"%s %o %s %s %d" % (kind, st.st_mode & 0o7777, owner.encode(), group.encode(), st.st_size)
    if kind != b"file":
        return s + b" -\n"
    if st.st_size <= int(limit):
        with open(path, "rb") as f:
            return s + b" +\nC:" + b64encode(f.read()) + b"\n"
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return s + b" " + h.hexdigest().encode() + b"\n"


out.write(b"ONIGIRAZU-READY P\n")
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
        if mode == b"P":
            try:
                so, se, rc = probe(command), b"", 0
            except Exception as err:
                so, se, rc = b"", str(err).encode() + b"\n", 1
            out.write(b"ONIGIRAZU %d %d %d\n" % (rc, len(so), len(se)) + so + se)
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
