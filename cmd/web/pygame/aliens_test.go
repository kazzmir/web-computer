package pygame

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kazzmir/web-computer/gpython/py"
	"github.com/kazzmir/web-computer/scripts"
)

func TestRepeatingPCM(t *testing.T) {
	data, err := io.ReadAll(&repeatingPCM{data: []byte{1, 2}, repeats: 2})
	if err != nil || !bytes.Equal(data, []byte{1, 2, 1, 2, 1, 2}) {
		t.Fatalf("finite loop = %v, %v", data, err)
	}
	data = make([]byte, 7)
	if _, err := io.ReadFull(&repeatingPCM{data: []byte{3, 4}, repeats: -1}, data); err != nil || !bytes.Equal(data, []byte{3, 4, 3, 4, 3, 4, 3}) {
		t.Fatalf("infinite loop = %v, %v", data, err)
	}
}

func testAliens(t *testing.T) {
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	system := &testSystem{}
	if err := RegisterPygameModule(system, ctx); err != nil {
		t.Fatal(err)
	}
	run := func(src, filename string, module interface{}) (*py.Module, error) {
		code, err := py.Compile(src, filename, "exec", 0, false)
		if err != nil {
			return nil, err
		}
		return py.RunCode(ctx, code, filename, module)
	}
	script, err := scripts.Load("aliens.py")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := run(string(script), "python/aliens.py", "aliens_test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = run(`
from pygame.sprite import Sprite, GroupSingle
assert Sprite is pg.sprite.Sprite and GroupSingle is pg.sprite.GroupSingle
assert Player.__annotations__['images'].__args__ == (pg.Surface,)
assert pg.image.get_extended()
assert pg.display.mode_ok((640, 480), 0, 32) == 32
screen = pg.display.set_mode((640, 480), 0, 32)
Player.images = [load_image('player1.gif')]
Player.images.append(pg.transform.flip(Player.images[0], 1, 0))
Explosion.images = [load_image('explosion1.gif')]
Explosion.images.append(pg.transform.flip(Explosion.images[0], 1, 1))
Alien.images = [load_image(name) for name in ('alien1.gif', 'alien2.gif', 'alien3.gif')]
Bomb.images = [load_image('bomb.gif')]
Shot.images = [load_image('shot.gif')]

# Bidirectional memberships, replacement, kill, and mutation during updates.
a, b = pg.sprite.Group(), pg.sprite.Group()
s = pg.sprite.Sprite(a, b)
assert s.alive() and len(s.groups()) == 2
assert s in a and a.has(s)
cloned = a.copy()
assert s in cloned
cloned.empty()
single = pg.sprite.GroupSingle(s)
s2 = pg.sprite.Sprite()
single.add(s2)
assert single.sprite is s2 and s in a and single not in s.groups(), 'single replacement'
s.kill()
assert not s.alive(), 'kill alive'
assert len(a) == 0 and len(b) == 0, 'kill group lengths'
assert not a and not b, 'empty group truth'
single.sprite = None
assert len(single) == 0, 'single clear length'
assert not single, 'single clear bool'
assert not s2.alive(), 'single clear membership'
all = pg.sprite.RenderUpdates()
player = Player(all)
assert player.rect.midbottom == SCREENRECT.midbottom
for i in range(100): player.move(1)
assert player.rect.right == SCREENRECT.right
for i in range(100): player.move(-1)
assert player.rect.left == SCREENRECT.left
oversized = pg.Rect(-50, -50, 700, 500).clamp(SCREENRECT)
assert oversized.center == SCREENRECT.center

alien = Alien(a, all)
alien.rect.right = SCREENRECT.right
alien.facing = Alien.speed
bottom = alien.rect.bottom
alien.update()
assert alien.facing == -Alien.speed and alien.rect.top == bottom+1
assert SCREENRECT.contains(alien.rect)
shot = Shot(player.gunpos(), b, all)
for i in range(50): shot.update()
assert not shot.alive() and not b
explosions = pg.sprite.Group()
e1, e2 = Explosion(player, explosions), Explosion(player, explosions)
e1.life = e2.life = 1
explosions.update()
assert not explosions and not e1.alive() and not e2.alive()
bomb = Bomb(alien, explosions, b, all)
bomb.rect.bottom = 469
bomb.update()
assert not bomb.alive() and len(explosions) == 1

# Collision results retain identity keys and remove victims from every group.
a.empty()
alien = Alien(a, all)
shot = Shot((100, 100), b, all)
alien.rect.center = shot.rect.center
collisions = pg.sprite.groupcollide(a, b, 1, 1)
assert collisions.keys() == [alien] and collisions[alien] == [shot]
assert not alien.alive() and not shot.alive() and not a and not b
alien = Alien(a, all)
assert pg.sprite.spritecollide(player, a, 1, lambda actor, target: True) == [alien]
assert not alien.alive()

# Dirty clears erase old and killed sprites, not just draw new positions.
background = pg.Surface((8, 4))
background.fill('blue')
small = pg.Surface((8, 4))
small.blit(background, (0, 0))
render = pg.sprite.RenderUpdates()
s = pg.sprite.Sprite(render)
s.image = pg.Surface((2, 2))
s.image.fill('red')
s.rect = s.image.get_rect(topleft=(1, 1))
render.draw(small)
s.rect.move_ip(3, 0)
render.clear(small, background)
dirty = render.draw(small)
assert len(dirty) == 2
assert small.get_at((1, 1)) == pg.Color('blue')
assert small.get_at((4, 1)) == pg.Color('red')
s.kill()
render.clear(small, background)
render.draw(small)
assert small.get_at((4, 1)) == pg.Color('blue')
backup = small.copy()
small.fill('white')
assert backup.get_at((4, 1)) == pg.Color('blue')
SCORE = 0
score = Score(all)
assert score.font.get_italic()
old_image = score.image
SCORE = 2
score.update()
assert score.lastscore == 2 and score.image is not old_image
try:
    load_image('missing-aliens-image.gif')
except SystemExit:
    assert 'missing-aliens-image.gif' in pg.get_error()
else:
    assert False

# Deterministic main loop: movement, held/released fire, two-shot cap,
# bullet hits, score/explosions, fullscreen copies, and Escape shutdown.
players, shots, spawned = [], [], []
real_player_init, real_shot_init, real_alien_init = Player.__init__, Shot.__init__, Alien.__init__
def track_player(self, *groups):
    real_player_init(self, *groups)
    players.append(self)
def track_shot(self, *args):
    real_shot_init(self, *args)
    shots.append(self)
def track_alien(self, *groups):
    real_alien_init(self, *groups)
    self.facing = 0
    self.rect.midbottom = (331, 200)
    spawned.append(self)
Player.__init__, Shot.__init__, Alien.__init__ = track_player, track_shot, track_alien
random.random = lambda: 0.5
keys = [False] * len(pg.key.get_pressed())
pg.key.get_pressed = lambda: keys
calls = 0
class Event:
    def __init__(self, kind, key=None): self.type, self.key = kind, key

def events():
    global calls
    calls += 1
    for key in (pg.K_LEFT, pg.K_RIGHT, pg.K_SPACE): keys[key] = False
    if calls == 1: keys[pg.K_RIGHT] = True
    if calls == 4: keys[pg.K_LEFT] = True
    if calls in (1, 2, 4, 6, 8, 21): keys[pg.K_SPACE] = True
    if calls in (9, 10): return [Event(pg.KEYDOWN, pg.K_f)]
    if calls == 24: return [Event(pg.KEYDOWN, pg.K_ESCAPE)]
    return []
pg.event.get = events
SCORE = 0
main()
assert calls == 24
assert len(players) == 1 and players[0].alive()
assert len(shots) == 3
assert len(spawned) == 1 and not spawned[0].alive()
assert SCORE == 1
pg.quit()
`, "aliens assertions", mod)
	if err != nil {
		py.TracebackDump(err)
		t.Fatalf("aliens integration: %v", err)
	}
	if system.frames != 26 {
		t.Errorf("got %d frames, want 26", system.frames)
	}
}

func testMusic(t *testing.T) {
	// Null ALSA is explicit audio configuration, permitting headless playback
	// state/fade tests without depending on a physical sound device.
	config := filepath.Join(t.TempDir(), "asound.conf")
	if err := os.WriteFile(config, []byte("pcm.!default { type null }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALSA_CONFIG_PATH", config)
	ctx := py.NewContext(py.DefaultContextOpts())
	defer ctx.Close()
	if err := RegisterPygameModule(&testSystem{}, ctx); err != nil {
		t.Fatal(err)
	}
	code, err := py.Compile(`
import pygame as pg
import time
pg.mixer.pre_init(44100, 32, 2, 1024)
assert pg.mixer.get_init() == (44100, -16, 2)
pg.mixer.music.load('data/house_lo.wav')
pg.mixer.music.play(-1)
assert pg.mixer.music.get_busy()
pg.mixer.music.fadeout(15)
deadline = time.time() + 1
while pg.mixer.music.get_busy() and time.time() < deadline:
    time.sleep(0.01)
assert not pg.mixer.music.get_busy()
pg.mixer.music.play(-1)
pg.mixer.music.stop()
assert not pg.mixer.music.get_busy()
pg.mixer.music.play(-1)
pg.mixer.music.fadeout(1000)
pg.quit()
assert not pg.mixer.music.get_busy()
`, "music assertions", "exec", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := py.RunCode(ctx, code, "music assertions", nil); err != nil {
		t.Fatal(err)
	}
}
