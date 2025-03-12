def foo(z, a=2):
    return z + a * 3

x = 3
y = 8
z = x * y
print(z)
print(foo(3))
print(foo(4, a=5))
