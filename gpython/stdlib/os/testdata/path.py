import os.path
import os.path as alias
from os.path import join
from os import path
assert alias is path
assert join('a', 'b') == 'a/b'
assert path is os.path
assert path.join('a', 'b', '') == 'a/b/'
assert path.join('a', '/b', 'c') == '/b/c'
assert path.join('', 'a') == 'a'
assert path.split('/a/b/') == ('/a/b', '')
assert path.split('/') == ('/', '')
assert path.split('//a') == ('//', 'a')
assert path.split('file') == ('', 'file')
assert path.split('') == ('', '')
assert path.basename('/a/b') == 'b'
assert path.dirname('/a/b') == '/a'
assert path.normpath('a/../b//./c') == 'b/c'
assert path.normpath('') == '.'
assert path.normpath('//a/../b') == '//b'
assert path.abspath('.') == os.getcwd()
assert path.isabs(path.abspath(__file__))
assert path.isdir(os.getcwd())
assert not path.isfile(os.getcwd())
assert not path.exists('missing-path-test-182994')
try:
    path.isabs()
except TypeError:
    pass
else:
    assert False
