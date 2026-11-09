def prepare_class(name, bases=(), kwds=None):
    keywords = {} if kwds is None else dict(kwds)
    if 'metaclass' in keywords:
        meta = keywords['metaclass']
        del keywords['metaclass']
    elif bases:
        meta = type(bases[0])
    else:
        meta = type
    if isinstance(meta, type):
        for base in bases:
            candidate = type(base)
            if issubclass(meta, candidate): continue
            if issubclass(candidate, meta): meta = candidate
            else: raise TypeError('metaclass conflict')
    namespace = meta.__prepare__(name, bases, **keywords) if hasattr(meta, '__prepare__') else {}
    return meta, namespace, keywords

def new_class(name, bases=(), kwds=None, exec_body=None):
    meta, namespace, keywords = prepare_class(name, bases, kwds)
    if exec_body is not None: exec_body(namespace)
    return meta(name, bases, namespace, **keywords)
