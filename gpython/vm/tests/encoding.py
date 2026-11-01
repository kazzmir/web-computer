doc = 'text encoding and byte-source decoding'
assert 'Ç'.encode('Latin-1') == b'\xc7'
assert bytes('Ç', 'Latin-1') == b'\xc7'
assert 'Питон'.encode() == b'\xd0\x9f\xd0\xb8\xd1\x82\xd0\xbe\xd0\xbd'
assert 'x蟒y'.encode('ascii', 'replace') == b'x?y'
assert 'x蟒y'.encode('ascii', 'ignore') == b'xy'
namespace = {}
exec(compile(b'# coding: Latin-1\nu = "\xc7"\n', 'source.py', 'exec'), namespace)
assert namespace['u'] == 'Ç'
namespace = {}
exec(b'# coding: Latin-1\nu = "\xc7"\n', namespace)
assert namespace['u'] == 'Ç'
assert eval(b'# coding: Latin-1\n"\xc7"') == 'Ç'
# str input has already been decoded, so its encoding cookie is not applied.
namespace = {}
exec(compile('# coding: Latin-1\nu = "Ç"\n', 'source.py', 'exec'), namespace)
assert namespace['u'] == 'Ç'
doc = 'finished'
