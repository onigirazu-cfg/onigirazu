# Bootstrap a machine, build an image

## bootstrap

A fresh machine is reachable as root (or the image's user) with a password or a key. After
`onigirazu bootstrap` it is a managed host: a deploy user with your public key and passwordless
sudo, password logins off if you ask.

```bash
onigirazu bootstrap 192.168.1.50 --ask-pass                                   # root + password
onigirazu bootstrap new-vm:2222 -u ubuntu --key ~/.ssh/cloud.pem --new-user deploy
onigirazu bootstrap 10.0.0.9 --ask-pass --disable-password-auth --keep-root-login=false
onigirazu bootstrap 10.0.0.9 --password-env BOOTSTRAP_PASSWORD --check        # show the changes only
```

What runs, as one generated playbook with `become`: the admin group is found (`sudo`, else
`wheel`, or `--sudo-group`); the user is created with `--shell` (default `/bin/bash`) and added
to it; `--pubkey` (default `~/.ssh/id_ed25519.pub`, then `id_rsa.pub`) goes to its
`authorized_keys`; `/etc/sudoers.d/90-<user>` gives `NOPASSWD: ALL` (checked with `visudo`);
with `--disable-password-auth`, `PasswordAuthentication no`; with `--keep-root-login=false`,
`PermitRootLogin prohibit-password` (both checked with `sshd -t`, sshd reloaded). Then
onigirazu connects as the new user with the matching private key and runs `sudo -n true`;
only when that works it prints the inventory line to add. The password is read from the
terminal or an environment variable, never from the command line.

## image build

```bash
onigirazu image build app.yml --from ubuntu:24.04 --tag registry.example.com/app:1.4
onigirazu image build app.yml --from debian:12 --tag app:dev \
  --cmd '["/usr/sbin/nginx", "-g", "daemon off;"]' --label team=web --runtime podman --push
```

A container starts from `--from` (kept alive with a shell loop), the playbook runs inside it
over the `docker`/`podman` connection (one host, as root, no SSH, no Python), and the container
is committed as `--tag` with `--cmd`, `--entrypoint` and `--label`s; `onigirazu.playbook` and
`onigirazu.built` labels are added. `--push` pushes it, `--keep` leaves the build container
for a look. The playbook is an ordinary one — the roles you apply to servers build the image
too; `onigirazu_image_build: true` is set on the host for tasks that should differ in an image
(no services to start, say):

```yaml
- hosts: all
  roles: [common, app]
  tasks:
    - service: {name: app, state: started}
      when: not (onigirazu_image_build | default(false))
```

`-e`, `--tags` and `--become` are passed to the run. The base image needs `sh` and `sleep`
(every Debian, Ubuntu, Alpine, Fedora image has them; a `scratch`-based image has not).
