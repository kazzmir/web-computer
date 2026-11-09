doc = 'lone surrogate code points are preserved, not replaced'
s = '\ud800x\udfff\ufffd'
assert len(s) == 4
assert s[0] == chr(0xd800)
assert ord(s[0]) == 0xd800
assert ord(s[2]) == 0xdfff
assert ord(s[3]) == 0xfffd
assert [ord(c) for c in s] == [0xd800, 120, 0xdfff, 0xfffd]
assert [ord(c) for c in s[::-1]] == [0xfffd, 0xdfff, 120, 0xd800]
assert s[::2] == '\ud800\udfff'
assert repr('\ud800') == "'\\ud800'"
assert '\ud800' != '\ufffd'
for text in ('\ud800', '\udfff', 'x\ud800y'):
    try: text.encode('utf-8')
    except UnicodeEncodeError: pass
    else: assert False
assert '\ud800'.encode('utf-8', 'surrogatepass') == b'\xed\xa0\x80'
assert 'x\ud800y'.encode('utf-8', 'ignore') == b'xy'
assert 'x\ud800y'.encode('utf-8', 'replace') == b'x?y'
import xml
assert xml.is_valid_text('\ufffd')
assert not xml.is_valid_text('\ud800')
assert not xml.is_valid_name('name\ud800')
assert xml.is_valid_name('name\U000EFFFF')
assert not xml.is_valid_name('name\U000F0000')
doc = 'finished'
