import pygame

a, b = pygame.init()
print("Successes", a)
print("Failures", b)

screen = pygame.display.set_mode((720, 480))

while True:
    screen.fill(0)
    pygame.display.update()
