from __future__ import barry_as_FLUFL
doc = 'future comparison operators and compiler-flag propagation'
import __future__
flag = __future__.CO_FUTURE_BARRY_AS_BDFL
assert flag == 0x400000
def compare(a, b): return a <> b
assert compare(1, 2)
assert not compare(1, 1)
assert compare.__code__.co_flags & flag
assert eval(compile('1 <> 2', '<flag>', 'eval', flag))
code = compile('from __future__ import barry_as_FLUFL\nanswer = 1 <> 2', '<directive>', 'exec')
assert code.co_flags & flag
namespace = {}
exec(code, namespace)
assert namespace['answer'] is True
for text in ('"from __future__ import barry_as_FLUFL"\n1 <> 2', '# from __future__ import barry_as_FLUFL\n1 <> 2', 'from .__future__ import barry_as_FLUFL\n1 <> 2'):
    try:
        compile(text, '<not a future>', 'exec', 0, True)
    except SyntaxError as error:
        assert '1 <> 2' in error.text
        assert error.filename == '<not a future>'
    else:
        assert False
doc = 'finished'
