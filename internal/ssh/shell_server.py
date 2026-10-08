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


def write(request):
    # "MODE OWNER GROUP\npath\ncontent": MODE octal or "-" (keep, 0644 for a
    # new file), OWNER/GROUP a name, an id or "-" (keep); written next to the
    # file and moved over it, as install(1) does
    head, _, rest = request.partition(b"\n")
    path, _, data = rest.partition(b"\n")
    mode, owner, group = head.split(b" ")
    d = os.path.dirname(path) or b"."
    os.makedirs(d, exist_ok=True)
    try:
        st = os.stat(path)
    except FileNotFoundError:
        st = None
    m = int(mode, 8) if mode != b"-" else (stat.S_IMODE(st.st_mode) if st else 0o644)

    def ident(name, lookup, current):
        if name == b"-":
            return current
        n = name.decode()
        return int(n) if n.isdigit() else lookup(n)

    uid = ident(owner, lambda n: pwd.getpwnam(n).pw_uid, st.st_uid if st else -1)
    gid = ident(group, lambda n: grp.getgrnam(n).gr_gid, st.st_gid if st else -1)
    fd, tmp = tempfile.mkstemp(prefix=b".onigirazu-", dir=d)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(data)
        os.chmod(tmp, m)
        if uid != -1 or gid != -1:
            os.chown(tmp, uid, gid)
        os.rename(tmp, path)
    except BaseException:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


out.write(b"ONIGIRAZU-READY P W\n")
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
        if mode == b"Q":
            # several probes in one answer: "limit\npath\npath...", each
            # record followed by "\x1e\n"
            limit, _, paths = command.partition(b"\n")
            parts = []
            for path in paths.split(b"\n"):
                try:
                    parts.append(probe(limit + b" " + path))
                except Exception as err:
                    parts.append(b"error " + str(err).encode() + b"\n")
            so = b"\x1e\n".join(parts) + b"\x1e\n"
            out.write(b"ONIGIRAZU 0 %d 0\n" % len(so) + so)
            out.flush()
            continue
        if mode == b"W":
            try:
                write(command)
                so, se, rc = b"", b"", 0
            except Exception as err:
                so, se, rc = b"", str(err).encode() + b"\n", 1
            out.write(b"ONIGIRAZU %d %d %d\n" % (rc, len(so), len(se)) + so + se)
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
