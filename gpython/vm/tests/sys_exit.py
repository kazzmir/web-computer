doc = 'sys.exit raises SystemExit and preserves its status'
import sys
for status in (None, 0, 3, 'goodbye'):
    finished = False
    try:
        try:
            sys.exit(status)
        finally:
            finished = True
    except SystemExit as error:
        assert error.code == status, 'exit status'
        assert finished, 'finally cleanup'
    else:
        assert False
try:
    sys.exit()
except SystemExit as error:
    assert error.code is None
else:
    assert False
doc = 'finished'
