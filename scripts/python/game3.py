import pygame

a, b = pygame.init()
print("{0} successes and {1} failures".format(successes, failures))

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
        elif event.type == pygame.KEYDOWN:
            if event.key == pygame.K_UP:
                rect.move_ip(0, -2)
            elif event.key == pygame.K_DOWN:
                rect.move_ip(0, 2)
            elif event.key == pygame.K_LEFT:
                rect.move_ip(-2, 0)
            elif event.key == pygame.K_RIGHT:
                rect.move_ip(2, 0)

    # rect.move_ip(0, 2)
    screen.fill(BLACK)
    screen.blit(image, rect)
    pygame.display.update()
