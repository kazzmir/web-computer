# Python-facing objects; pixel operations and input are backed by Ebitengine.
FULLSCREEN = -2147483648
SCALED = 512
HWSURFACE = 1
DOUBLEBUF = 1073741824
OPENGL = 2
RESIZABLE = 16
GL_CONTEXT_MAJOR_VERSION = 17
GL_CONTEXT_MINOR_VERSION = 18
GL_CONTEXT_PROFILE_MASK = 21
GL_CONTEXT_PROFILE_CORE = 1
GL_CONTEXT_PROFILE_COMPATIBILITY = 2
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
    def _get_midbottom(self): return self.centerx, self.bottom
    def _set_midbottom(self, value): self.centerx, self.bottom = value
    midbottom = property(_get_midbottom, _set_midbottom)

    def clamp_ip(self, bounds):
        if self.width >= bounds.width: self.centerx = bounds.centerx
        else: self.x = max(bounds.left, min(self.x, bounds.right-self.width))
        if self.height >= bounds.height: self.centery = bounds.centery
        else: self.y = max(bounds.top, min(self.y, bounds.bottom-self.height))
    def clamp(self, bounds):
        rect = self.copy()
        rect.clamp_ip(bounds)
        return rect

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
    def copy(self): return _wrap(_copy(self._pixels))
    def get_size(self): return _size(self._pixels)
    def get_bitsize(self): return 32
    def get_width(self): return self.get_size()[0]
    def get_height(self): return self.get_size()[1]
    def get_rect(self, **kwargs):
        r = Rect((0, 0), self.get_size())
        for key in kwargs: setattr(r, key, kwargs[key])
        return r
    def fill(self, color, rect=None):
        if rect is not None:
            rect = rect if isinstance(rect, Rect) else Rect(rect)
            rect = (rect.x, rect.y, rect.w, rect.h)
        return Rect(_fill(self._pixels, color, rect))
    def blit(self, source, dest, area=None):
        if isinstance(dest, Rect): dest = dest.topleft
        if area is not None:
            area = area if isinstance(area, Rect) else Rect(area)
            area = (area.x, area.y, area.w, area.h)
        return Rect(_blit(self._pixels, source._pixels, int(dest[0]), int(dest[1]), area))
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
    def __init__(self):
        self.surface = None
        self._gl_attributes = [0] * 32
    def init(self): pass
    def gl_set_attribute(self, attribute, value): self._gl_attributes[int(attribute)] = int(value)
    def gl_get_attribute(self, attribute): return self._gl_attributes[int(attribute)]
    def mode_ok(self, size, flags=0, depth=0): return 32 if size[0] > 0 and size[1] > 0 else 0
    def set_icon(self, surface): _icon(surface._pixels)
    def set_mode(self, size, flags=0, depth=0):
        self.surface = Surface(size)
        _set_mode(int(size[0]), int(size[1]), not not (flags & FULLSCREEN))
        _gl_mode(self.surface._pixels, not not (flags & OPENGL))
        return self.surface
    def get_surface(self): return self.surface
    def set_caption(self, caption): _caption(caption)
    def flip(self): _present(self.surface._pixels)
    def update(self, rect=None): self.flip()
display = _Display()

class _Image:
    def get_extended(self): return True
    def load(self, filename): return _wrap(_load(filename))
    def save(self, surface, filename): _save(surface._pixels, filename)
image = _Image()

class _Transform:
    def _scale(self, surface, size, dest_surface, smooth):
        w, h = int(size[0]), int(size[1])
        if dest_surface is None: return _wrap(_scale(surface._pixels, w, h, smooth))
        _scale_to(surface._pixels, dest_surface._pixels, w, h, smooth)
        return dest_surface
    def scale(self, surface, size, dest_surface=None): return self._scale(surface, size, dest_surface, False)
    def smoothscale(self, surface, size, dest_surface=None): return self._scale(surface, size, dest_surface, True)
    def scale2x(self, surface, dest_surface=None):
        pixels = _scale2x(surface._pixels, None if dest_surface is None else dest_surface._pixels)
        return _wrap(pixels) if dest_surface is None else dest_surface
    def flip(self, surface, x, y): return _wrap(_flip(surface._pixels, not not x, not not y))
    def rotate(self, surface, angle): return _wrap(_rotate(surface._pixels, angle))
transform = _Transform()

class _Mouse:
    def get_pos(self): return _mouse_pos()
    def set_visible(self, visible): return _mouse_visible(not not visible)
mouse = _Mouse()

class _Clock:
    def __init__(self):
        self._last = _ticks()
        self._fps_start = self._last
        self._frames = 0
        self._fps = 0.0
    def get_fps(self): return self._fps
    def tick(self, fps=0):
        _tick(int(fps))
        now = _ticks()
        elapsed = now - self._last
        self._last = now
        self._frames += 1
        if self._frames >= 10:
            period = now - self._fps_start
            self._fps = 1000.0 * self._frames / period if period else 0.0
            self._frames = 0
            self._fps_start = now
        return elapsed
class _Time:
    def wait(self, milliseconds): return _wait(int(milliseconds))
    def get_ticks(self): return _ticks()
time = _Time()
time.Clock = _Clock

class _Event: pass
class _Events:
    def get(self):
        events = []
        for kind, key, window in _events():
            e = _Event()
            e.type = kind
            e.window = window
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
    def __init__(self, filename, size):
        self._face = _font(filename, int(size))
        self._italic = False
    def set_italic(self, value):
        self._italic = not not value
        _font_italic(self._face, self._italic)
    def get_italic(self): return self._italic
    def render(self, message, antialias, color):
        return _wrap(_render(self._face, message, antialias, color))
class _Fonts: pass
font = _Fonts()
font.Font = _Font

class _Sound:
    def __init__(self, filename): self._sound = _sound(filename)
    def play(self): _play(self._sound)
class _Music:
    def load(self, filename): _music_load(filename)
    def play(self, loops=0): _music_play(int(loops))
    def stop(self): _music_stop()
    def fadeout(self, milliseconds): _music_fadeout(int(milliseconds))
    def get_busy(self): return _music_busy()

class _Mixer:
    def __init__(self): self.music = _Music()
    def pre_init(self, frequency=44100, size=-16, channels=2, buffer=512):
        if frequency <= 0 or channels <= 0 or buffer <= 0: raise ValueError('invalid mixer settings')
    def get_init(self): return _mixer_init()
mixer = _Mixer()
mixer.Sound = _Sound

def get_sdl_version(): return (2, 0, 0)
def get_error(): return _get_error()

def init(): return 3, 0
def quit():
    _sdl_quit()
    _audio_quit()
    mouse.set_visible(True)
