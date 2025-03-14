import pygame

a, b = pygame.init()
print("Successes", a)
print("Failures", b)

BLACK = (0, 0, 0)
WHITE = (255, 255, 255)
clock = pygame.time.Clock()
FPS = 60

rect = pygame.Rect((0, 0), (32, 32))
screen = pygame.display.set_mode((720, 480))
image = pygame.Surface((32, 32))
image.fill(WHITE)

while True:
    clock.tick(FPS)
    for event in pygame.event.get():
        if event.type == pygame.QUIT:
            quit()

    rect.move_ip(0, 2)
    screen.fill(BLACK)
    screen.blit(image, rect)
    pygame.display.update()
