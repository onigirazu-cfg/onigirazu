# Stdout callback for onigirazu's Ansible bridge: writes the result of the
# one task of the run to $ONIGIRAZU_BRIDGE_RESULT as JSON.
import json
import os

from ansible.plugins.callback import CallbackBase

try:
    from ansible.module_utils.common.json import AnsibleJSONEncoder as Encoder
except ImportError:  # older ansible-core
    Encoder = json.JSONEncoder


class CallbackModule(CallbackBase):
    CALLBACK_VERSION = 2.0
    CALLBACK_TYPE = "stdout"
    CALLBACK_NAME = "onigirazu_bridge"

    def __init__(self):
        super(CallbackModule, self).__init__()
        self.out = None

    def _keep(self, status, result):
        self.out = {"status": status, "result": dict(result._result)}

    def v2_runner_on_ok(self, result):
        self._keep("changed" if result._result.get("changed") else "ok", result)

    def v2_runner_on_failed(self, result, ignore_errors=False):
        self._keep("failed", result)

    def v2_runner_on_skipped(self, result):
        self._keep("skipped", result)

    def v2_runner_on_unreachable(self, result):
        self._keep("unreachable", result)

    def v2_playbook_on_stats(self, stats):
        if self.out is None:
            return
        with open(os.environ["ONIGIRAZU_BRIDGE_RESULT"], "w") as f:
            try:
                json.dump(self.out, f, cls=Encoder)
            except TypeError:
                f.seek(0)
                f.truncate()
                json.dump(self.out, f, default=str)
