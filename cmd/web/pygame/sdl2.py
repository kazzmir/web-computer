# SDL2 rendering compatibility over a single Ebitengine virtual desktop.
import pygame as pg

class Window:
    id = property(lambda self: self._id)
    def __init__(self, title='pygame', size=(640, 480), position=None, resizable=False, always_on_top=False):
        self._id = pg._sdl_window_new(self, title, size, not not resizable, not not always_on_top)
        if position is not None: self.position = position
    def _get_title(self): return pg._sdl_window_get(self._id, 'title')
    def _set_title(self, value): pg._sdl_window_set(self._id, 'title', str(value))
    title = property(_get_title, _set_title)
    def _get_opacity(self): return pg._sdl_window_get(self._id, 'opacity')
    def _set_opacity(self, value): pg._sdl_window_set(self._id, 'opacity', value)
    opacity = property(_get_opacity, _set_opacity)
    def _get_size(self): return pg._sdl_window_get(self._id, 'size')
    def _set_size(self, value): pg._sdl_window_set(self._id, 'size', value)
    size = property(_get_size, _set_size)
    def _get_position(self): return pg._sdl_window_get(self._id, 'position')
    def _set_position(self, value): pg._sdl_window_set(self._id, 'position', value)
    position = property(_get_position, _set_position)
    def set_icon(self, surface): pg._sdl_window_icon(self._id, surface._pixels)
    def set_fullscreen(self, desktop=False): pg._sdl_window_fullscreen(self._id, True)
    def set_windowed(self): pg._sdl_window_fullscreen(self._id, False)
    def focus(self): pg._sdl_window_focus(self._id)
    def destroy(self): pg._sdl_window_destroy(self._id)

class Renderer:
    def __init__(self, window, index=-1, accelerated=-1, vsync=False, target_texture=False):
        if index not in (-1, 0): raise pg.error('only the Ebitengine renderer is available')
        if target_texture: raise pg.error('render-to-texture targets are not implemented')
        self.window = window
        self._id = pg._sdl_renderer_new(window._id)
    def _get_color(self): return pg._sdl_renderer_color(self._id, None)
    def _set_color(self, color): pg._sdl_renderer_color(self._id, color)
    draw_color = property(_get_color, _set_color)
    def clear(self): pg._sdl_renderer_shape(self._id, 'clear', None)
    def draw_line(self, a, b): pg._sdl_renderer_shape(self._id, 'line', (a, b))
    def draw_point(self, pos): pg._sdl_renderer_shape(self._id, 'point', pos)
    def draw_rect(self, rect):
        r = rect if isinstance(rect, pg.Rect) else pg.Rect(rect)
        pg._sdl_renderer_shape(self._id, 'rect', (r.x, r.y, r.w, r.h))
    def fill_rect(self, rect):
        r = rect if isinstance(rect, pg.Rect) else pg.Rect(rect)
        pg._sdl_renderer_shape(self._id, 'fill', (r.x, r.y, r.w, r.h))
    def present(self): pg._sdl_present(self._id)
    def to_surface(self): return pg._wrap(pg._sdl_renderer_read(self._id))

class Texture:
    def __init__(self, renderer, size, streaming=False):
        self.renderer = renderer
        self._id = pg._sdl_texture_new(renderer._id, size, not not streaming)
    @classmethod
    def from_surface(cls, renderer, surface):
        texture = cls(renderer, surface.get_size())
        texture.update(surface)
        return texture
    def _get_width(self): return pg._sdl_texture_size(self._id)[0]
    def _get_height(self): return pg._sdl_texture_size(self._id)[1]
    width = property(_get_width)
    height = property(_get_height)
    def update(self, surface): pg._sdl_texture_update(self._id, surface._pixels)
    def draw(self, srcrect=None, dstrect=None):
        if srcrect is None: src = pg.Rect(0, 0, self.width, self.height)
        else: src = srcrect if isinstance(srcrect, pg.Rect) else pg.Rect(srcrect)
        if dstrect is not None:
            if not isinstance(dstrect, pg.Rect):
                if len(dstrect) == 2: dstrect = (dstrect[0], dstrect[1], src.w, src.h)
                dstrect = pg.Rect(dstrect)
            dstrect = (dstrect.x, dstrect.y, dstrect.w, dstrect.h)
        pg._sdl_texture_draw(self._id, (src.x, src.y, src.w, src.h), dstrect)

class Image:
    def __init__(self, texture, srcrect=None):
        self.texture = texture
        self.srcrect = srcrect
    def draw(self, dstrect=None): self.texture.draw(self.srcrect, dstrect)

class _Driver:
    name = 'ebitengine'
    flags = 2
    max_texture_width = 2048
    max_texture_height = 2048
    def __repr__(self): return 'RendererDriverInfo(name=ebitengine, accelerated=True, virtual_windows=True)'

def get_drivers(): return iter([_Driver()])
def messagebox(title, message, info=False, warn=False, error=False, buttons=('OK',), return_button=0, escape_button=0):
    return pg._sdl_messagebox(title, message, buttons, int(return_button), int(escape_button))
