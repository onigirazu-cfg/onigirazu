# Prints one JSON line per finished task, in the shape of onigirazu's
# task_end records, so compat/run.sh can compare the two tools.
import json

from ansible.plugins.callback import CallbackBase

DOCUMENTATION = """
name: compat_results
type: stdout
short_description: task results as JSON lines for compat/run.sh
description: One JSON object per finished task and host.
"""


class CallbackModule(CallbackBase):
    CALLBACK_VERSION = 2.0
    CALLBACK_TYPE = "stdout"
    CALLBACK_NAME = "compat_results"

    def _emit(self, result, success, skipped=False, ignored=False):
        task = result._task
        res = result._result
        msg = ""
        if not success:
            msg = res.get("msg", "")
        elif task.action in ("debug", "ansible.builtin.debug") and not skipped:
            if "msg" in res:
                msg = res["msg"] if isinstance(res["msg"], str) else json.dumps(res["msg"], sort_keys=True)
            else:
                parts = []
                for k, v in res.items():
                    if k.startswith("_") or k in ("changed", "failed"):
                        continue
                    parts.append("%s: %s" % (k, v if isinstance(v, str) else json.dumps(v, sort_keys=True)))
                msg = ", ".join(parts)
        # a failure inside a block with a rescue section is handled there
        parent = task._parent
        if not success and parent is not None and getattr(parent, "rescue", None):
            ignored = True
        self._display.display(json.dumps({
            "host": result._host.get_name(),
            "task": task.get_name(),
            "module": task.action.split(".")[-1],
            "success": success,
            "changed": bool(res.get("changed")) and not skipped,
            "skipped": skipped,
            "ignored": ignored,
            "msg": msg,
        }, sort_keys=True))

    def v2_runner_on_ok(self, result):
        self._emit(result, True)

    def v2_runner_on_failed(self, result, ignore_errors=False):
        self._emit(result, False, ignored=ignore_errors)

    def v2_runner_on_skipped(self, result):
        self._emit(result, True, skipped=True)

    def v2_runner_on_unreachable(self, result):
        self._emit(result, False)
