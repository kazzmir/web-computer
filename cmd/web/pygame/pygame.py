# Python-facing objects; pixel operations and input are backed by Ebitengine.
FULLSCREEN = -2147483648
SCALED = 512
RLEACCEL = 16384
QUIT = 256
KEYDOWN = 768
KEYUP = 769
MOUSEBUTTONDOWN = 1025
MOUSEBUTTONUP = 1026

class Rect:
    def __init__(self, *args):
        if len(args) == 1:
            args = args[0]
        if len(args) == 2:
            args = tuple(args[0]) + tuple(args[1])
        self.x, self.y, self.width, self.height = [int(v) for v in args]

    def _get_left(self): return self.x
    def _set_left(self, v): self.x = int(v)
    left = property(_get_left, _set_left)
    def _get_top(self): return self.y
    def _set_top(self, v): self.y = int(v)
    top = property(_get_top, _set_top)
    def _get_right(self): return self.x + self.width
    def _set_right(self, v): self.x = int(v) - self.width
    right = property(_get_right, _set_right)
    def _get_bottom(self): return self.y + self.height
    def _set_bottom(self, v): self.y = int(v) - self.height
    bottom = property(_get_bottom, _set_bottom)
    def _get_topleft(self): return self.x, self.y
    def _set_topleft(self, v): self.x, self.y = int(v[0]), int(v[1])
    topleft = property(_get_topleft, _set_topleft)
    def _get_centerx(self): return self.x + self.width // 2
    def _set_centerx(self, v): self.x = int(v) - self.width // 2
    centerx = property(_get_centerx, _set_centerx)
    def _get_centery(self): return self.y + self.height // 2
    def _set_centery(self, v): self.y = int(v) - self.height // 2
    centery = property(_get_centery, _set_centery)
    def _get_center(self): return self.centerx, self.centery
    def _set_center(self, v): self.centerx, self.centery = v
    center = property(_get_center, _set_center)
    def _get_size(self): return self.width, self.height
    def _set_size(self, v): self.width, self.height = int(v[0]), int(v[1])
    size = property(_get_size, _set_size)
    w = property(lambda self: self.width, lambda self, value: setattr(self, 'width', int(value)))
    h = property(lambda self: self.height, lambda self, value: setattr(self, 'height', int(value)))

    def copy(self): return Rect(self.x, self.y, self.width, self.height)

    def move(self, *args):
        r = Rect(self.x, self.y, self.width, self.height)
        r.move_ip(*args)
        return r

    def move_ip(self, *args):
        if len(args) == 1: args = args[0]
        self.x += int(args[0])
        self.y += int(args[1])

    def inflate(self, x, y):
        return Rect(self.x - int(int(x) / 2), self.y - int(int(y) / 2),
                    self.width + int(x), self.height + int(y))

    def contains(self, r):
        return self.left <= r.left and self.top <= r.top and self.right >= r.right and self.bottom >= r.bottom

    def colliderect(self, r):
        return self.width > 0 and self.height > 0 and r.width > 0 and r.height > 0 and self.left < r.right and self.right > r.left and self.top < r.bottom and self.bottom > r.top

class Surface:
    def __init__(self, size, flags=0, depth=0, _pixels=None):
        if _pixels is None:
            self._pixels = _new_surface(int(size[0]), int(size[1]))
        else:
            self._pixels = _pixels

    def convert(self): return self
    def convert_alpha(self): return self
    def get_size(self): return _size(self._pixels)
    def get_width(self): return self.get_size()[0]
    def get_height(self): return self.get_size()[1]
    def get_rect(self, **kwargs):
        r = Rect((0, 0), self.get_size())
        for key in kwargs: setattr(r, key, kwargs[key])
        return r
    def fill(self, color):
        _fill(self._pixels, color)
        return self.get_rect()
    def blit(self, source, dest):
        if isinstance(dest, Rect): dest = dest.topleft
        _blit(self._pixels, source._pixels, int(dest[0]), int(dest[1]))
        return Rect(dest, source.get_size())
    def get_at(self, pos): return _get_at(self._pixels, int(pos[0]), int(pos[1]))
    def set_colorkey(self, color, flags=0): _colorkey(self._pixels, color)
    def get_clip(self): return Rect(_get_clip(self._pixels))
    def set_clip(self, rect=None):
        if rect is not None:
            rect = rect if isinstance(rect, Rect) else Rect(rect)
            rect = (rect.x, rect.y, rect.w, rect.h)
        _set_clip(self._pixels, rect)
    def subsurface(self, rect):
        rect = rect if isinstance(rect, Rect) else Rect(rect)
        return _wrap(_subsurface(self._pixels, rect.x, rect.y, rect.w, rect.h))
    def scroll(self, dx=0, dy=0): _scroll(self._pixels, int(dx), int(dy))

def _wrap(pixels):
    return Surface(None, _pixels=pixels)

def Color(*args):
    return _color(args[0] if len(args) == 1 else args)

class _Display:
    def __init__(self): self.surface = None
    def init(self): pass
    def set_mode(self, size, flags=0):
        self.surface = Surface(size)
        _set_mode(int(size[0]), int(size[1]), not not (flags & FULLSCREEN))
        return self.surface
    def get_surface(self): return self.surface
    def set_caption(self, caption): _caption(caption)
    def flip(self): _present(self.surface._pixels)
    def update(self, rect=None): self.flip()
display = _Display()

class _Image:
    def load(self, filename): return _wrap(_load(filename))
image = _Image()

class _Transform:
    def _scale(self, surface, size, dest_surface, smooth):
        w, h = int(size[0]), int(size[1])
        if dest_surface is None: return _wrap(_scale(surface._pixels, w, h, smooth))
        _scale_to(surface._pixels, dest_surface._pixels, w, h, smooth)
        return dest_surface
    def scale(self, surface, size, dest_surface=None): return self._scale(surface, size, dest_surface, False)
    def smoothscale(self, surface, size, dest_surface=None): return self._scale(surface, size, dest_surface, True)
    def flip(self, surface, x, y): return _wrap(_flip(surface._pixels, x, y))
    def rotate(self, surface, angle): return _wrap(_rotate(surface._pixels, angle))
transform = _Transform()

class _Mouse:
    def get_pos(self): return _mouse_pos()
    def set_visible(self, visible): return _mouse_visible(not not visible)
mouse = _Mouse()

class _Sprite:
    def __init__(self): pass
    def update(self): pass

class _Group:
    def __init__(self, *sprites):
        self._sprites = []
        self.add(*sprites)
    def add(self, *sprites):
        for s in sprites:
            if isinstance(s, (tuple, list)):
                self.add(*s)
            elif s not in self._sprites:
                self._sprites.append(s)
    def update(self, *args):
        for s in self._sprites: s.update(*args)
    def draw(self, surface):
        return [surface.blit(s.image, s.rect) for s in self._sprites]

class _Sprites: pass
sprite = _Sprites()
sprite.Sprite = _Sprite
sprite.Group = _Group
sprite.RenderPlain = _Group

class _Clock:
    def __init__(self): self._last = _ticks()
    def tick(self, fps=0):
        _tick(int(fps))
        now = _ticks()
        elapsed = now - self._last
        self._last = now
        return elapsed
class _Time: pass
time = _Time()
time.Clock = _Clock

class _Event: pass
class _Events:
    def get(self):
        events = []
        for kind, key in _events():
            e = _Event()
            e.type = kind
            e.key = key
            e.button = key
            if kind == MOUSEBUTTONDOWN or kind == MOUSEBUTTONUP: e.pos = mouse.get_pos()
            events.append(e)
        return events
event = _Events()

class _Key:
    def __init__(self): self._repeat = (0, 0)
    def get_pressed(self): return _key_pressed()
    def get_repeat(self): return self._repeat
    def set_repeat(self, delay=0, interval=0):
        delay, interval = int(delay), int(interval)
        if delay < 0 or interval < 0: raise ValueError('repeat values must be nonnegative')
        if delay and not interval: interval = delay
        self._repeat = (delay, interval)
        _key_repeat(delay, interval)
key = _Key()

class _Draw:
    def polygon(self, surface, color, points):
        return Rect(_polygon(surface._pixels, color, points))
draw = _Draw()

class _Font:
    def __init__(self, filename, size): self._face = _font(filename, int(size))
    def render(self, message, antialias, color):
        return _wrap(_render(self._face, message, antialias, color))
class _Fonts: pass
font = _Fonts()
font.Font = _Font

class _Sound:
    def __init__(self, filename): self._sound = _sound(filename)
    def play(self): _play(self._sound)
class _Mixer:
    def get_init(self): return (44100, -16, 2)
mixer = _Mixer()
mixer.Sound = _Sound

def init(): return 3, 0
def quit():
    _audio_quit()
    mouse.set_visible(True)
