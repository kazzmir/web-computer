doc = 'power protocol, integer modular inverses, and gcd'
import math
assert math.gcd() == 0
assert math.gcd(-12, 18, 30) == 6
assert math.gcd(1 << 100, 1 << 80) == 1 << 80
try: math.gcd(2.5, 4)
except TypeError: pass
else: assert False

base, exponent, modulus = 3, -1, -11
assert pow(base, exponent, modulus) == -7
assert (base, exponent, modulus) == (3, -1, -11)
assert pow(2, -1, 1) == 0
assert pow(2, -1, -1) == 0
for call in (lambda: pow(2, 3, 0), lambda: pow(2, -1, 4)):
    try: call()
    except ValueError: pass
    else: assert False
for call in (lambda: pow(0, -1), lambda: pow(0.0, -0.5)):
    try: call()
    except ZeroDivisionError: pass
    else: assert False
assert pow(0.0, float('-inf')) == float('inf')

class Base:
    def __pow__(self, exponent, modulus=None): return exponent, modulus
    def __rpow__(self, base): return base
    def __ipow__(self, exponent): self.exponent = exponent; return self
class Derived(Base): pass
value = Derived()
assert pow(value, 3, 5) == (3, 5)
assert value ** 4 == (4, None)
assert None ** value is None
alias = value
value **= 7
assert value is alias and value.exponent == 7
doc = 'finished'
