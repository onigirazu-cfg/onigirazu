# E2E tests

Runs every case in `cases/` on disposable vSphere VMs cloned from the current
`[latest]` Ubuntu golden images, then deletes the VMs.

- Workflow `E2E`: by hand (optionally a subset of cases, or keep the VMs) and
  nightly at 05:00 UTC; only on self-hosted runners labelled `vsphere-e2e`.
- Cases are split over four shards (`E2E_SHARD=i/4`, every fourth case), each
  on its own VMs; runs of different branches do not wait for each other.
- VMs are named `tmp-e2e-onigirazu-<run>-<os>`, live in a dedicated folder and
  carry a "TEMPORARY" note with the run link and expiry time.
- `janitor.sh` runs hourly and deletes e2e VMs older than 3 hours from that
  folder, covering cancelled runs. `purge_vms` removes all of them at once;
  VMs of runs still in progress are kept.
- Access: each run generates an SSH key and passes it as
  `guestinfo.e2e_authorized_key`; the image creates user `e2e` on first boot.

## A case

`cases/<NN-name>/`:
- `playbook.yml` — applied to all VMs (`hosts: all`);
- `setup.sh` — optional; runs on each VM as root before the first apply;
- `verify.sh` — runs on each VM as root after the apply and must exit 0;
- `verify-local.sh` — optional; runs on the runner in the case directory with `HOST` set
  (for results that land on the control machine, e.g. fetch);
- `EXPECT_FAIL` — optional; the apply must fail (no verify, no second apply);
- `NOT_IDEMPOTENT` — optional; skips the second apply that must change nothing.

With `keep_vms` the run key and inventory stay on the runner in
`~/.cache/onigirazu-e2e/<run>`.

The playbook runs from a copy of its case directory, so relative paths work.

## Configuration

Repository secrets: `VSPHERE_SERVER`, `VSPHERE_USER`, `VSPHERE_PASSWORD`,
`E2E_DATACENTER`, `E2E_CLUSTER`, `E2E_HOST`, `E2E_DATASTORE`, `E2E_NETWORK`,
`E2E_FOLDER`, `E2E_LIBRARY`. Secrets rather than variables: Actions logs of a
public repository are public.
