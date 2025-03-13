import pygame

a, b = pygame.init()
print("Successes", a)
print("Failures", b)

BLACK = (0, 0, 0)
WHITE = (255, 255, 255)

screen = pygame.display.set_mode((720, 480))
image = pygame.Surface((32, 32))
image.fill(WHITE)
rect = 2

while True:
    screen.fill(BLACK)
    screen.blit(image, rect)
    pygame.display.update()
