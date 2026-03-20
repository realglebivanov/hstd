from collections import defaultdict

_restart_triggers = defaultdict(list)

def notify(svc, op):
    _restart_triggers[svc].append(op)
    return op

def changed(svc):
    return any(t.changed for t in _restart_triggers[svc])
