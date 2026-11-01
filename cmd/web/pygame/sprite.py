class _Sprite:
    def __init__(self, *groups):
        self._groups = []
        self.add(*groups)
    def add(self, *groups):
        for group in groups:
            if isinstance(group, (list, tuple)): self.add(*group)
            else: group.add(self)
    def remove(self, *groups):
        for group in groups:
            if isinstance(group, (list, tuple)): self.remove(*group)
            else: group.remove(self)
    def groups(self): return list(self._groups)
    def alive(self): return len(self._groups) > 0
    def kill(self):
        for group in list(self._groups): group.remove(self)
    def update(self, *args, **kwargs): pass

def _flatten_sprites(sprites):
    out = []
    for value in sprites:
        if isinstance(value, _Sprite): out.append(value)
        elif isinstance(value, _Group): out.extend(value.sprites())
        elif isinstance(value, (tuple, list)): out.extend(_flatten_sprites(value))
        else: raise TypeError('expected a Sprite or sprite collection')
    return out

class _Group:
    def __init__(self, *sprites):
        self._sprites = []
        self._drawn = []
        self.add(*sprites)
    def _add(self, s):
        if s not in self._sprites:
            self._sprites.append(s)
            s._groups.append(self)
    def add(self, *sprites):
        for s in _flatten_sprites(sprites): self._add(s)
    def remove(self, *sprites):
        for s in _flatten_sprites(sprites):
            if s in self._sprites:
                self._sprites.remove(s)
                s._groups.remove(self)
    def has(self, *sprites): return all([s in self._sprites for s in _flatten_sprites(sprites)])
    def sprites(self): return list(self._sprites)
    def empty(self): self.remove(*list(self._sprites))
    def copy(self): return type(self)(*self._sprites)
    def __len__(self): return len(self._sprites)
    def __iter__(self): return iter(self.sprites())
    def __contains__(self, s): return s in self._sprites
    def update(self, *args, **kwargs):
        # Updates may remove sprites or create new ones without skipping peers.
        for s in self.sprites(): s.update(*args, **kwargs)
    def clear(self, surface, background):
        for rect in self._drawn:
            if isinstance(background, Surface): surface.blit(background, rect, rect)
            else: background(surface, rect)
    def draw(self, surface):
        rects = [surface.blit(s.image, s.rect) for s in self.sprites()]
        self._drawn = [r.copy() for r in rects]
        return rects

class _RenderUpdates(_Group):
    def draw(self, surface):
        old = self._drawn
        new = _Group.draw(self, surface)
        return old + new

class _GroupSingle(_Group):
    def _add(self, s):
        if s not in self._sprites:
            self.empty()
            _Group._add(self, s)
    def _get_sprite(self): return self._sprites[0] if self._sprites else None
    def _set_sprite(self, s):
        self.empty()
        if s is not None: self.add(s)
    sprite = property(_get_sprite, _set_sprite)

# GPython's native dictionaries currently accept string keys only. This
# mapping preserves Sprite identity keys and the groupcollide mapping protocol.
class _SpriteCollisions:
    def __init__(self, pairs): self._pairs = pairs
    def keys(self): return [s for s, hits in self._pairs]
    def values(self): return [hits for s, hits in self._pairs]
    def items(self): return list(self._pairs)
    def __len__(self): return len(self._pairs)
    def __iter__(self): return iter(self.keys())
    def __contains__(self, key): return any([s is key for s, hits in self._pairs])
    def __getitem__(self, key):
        for s, hits in self._pairs:
            if s is key: return hits
        raise KeyError(key)
    def get(self, key, default=None):
        for s, hits in self._pairs:
            if s is key: return hits
        return default

def _spritecollide(actor, group, dokill, collided=None):
    hits = []
    for s in group.sprites():
        hit = actor.rect.colliderect(s.rect) if collided is None else collided(actor, s)
        if hit:
            hits.append(s)
            if dokill: s.kill()
    return hits

def _groupcollide(a, b, dokilla, dokillb, collided=None):
    pairs = []
    for s in a.sprites():
        hits = _spritecollide(s, b, dokillb, collided)
        if hits:
            pairs.append((s, hits))
            if dokilla: s.kill()
    return _SpriteCollisions(pairs)

class _Sprites: pass
sprite = _Sprites()
sprite.Sprite = _Sprite
sprite.Group = _Group
sprite.RenderPlain = _Group
sprite.RenderUpdates = _RenderUpdates
sprite.GroupSingle = _GroupSingle
sprite.spritecollide = _spritecollide
sprite.groupcollide = _groupcollide
