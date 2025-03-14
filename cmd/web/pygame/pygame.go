package pygame

import (
    "log"
    "fmt"
    "image/color"
    "time"
    // "reflect"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
    "github.com/go-python/gpython/py"
)

type PythonMethod = func (py.Object, py.Tuple) (py.Object, error)

type GraphicsSystem interface {
    Flip()
    AddDrawCall(func(*ebiten.Image))
    SetScreenSize(int, int)
}

type DisplaySurface struct {
    pyType *py.Type
}

func (surface *DisplaySurface) Type() *py.Type {
    return surface.pyType
}

type RegularSurface struct {
    Image *ebiten.Image
    pyType *py.Type
}

func (surface *RegularSurface) Type() *py.Type {
    return surface.pyType
}

type Rect struct {
    x1, y1, x2, y2 py.Int
    pyType *py.Type
}

func (rect *Rect) Type() *py.Type {
    return rect.pyType
}

type TimeClock struct {
    lastTime time.Time
    pyType *py.Type
}

func (clock *TimeClock) Type() *py.Type {
    return clock.pyType
}

func createTimeModule() *py.ModuleImpl {
    clock_tick := func (self py.Object, args py.Tuple) (py.Object, error) {
        var value py.Int
        err := py.LoadTuple(args, []interface{}{&value})
        if err != nil {
            return py.None, err
        }

        num, err := value.GoInt()
        if err != nil {
            return py.None, err
        }

        ebiten.SetTPS(num)

        clock := self.(*TimeClock)
        duration := time.Since(clock.lastTime)
        clock.lastTime = time.Now()

        // pygame also returns an int here
        return py.Int(duration / time.Millisecond), nil
    }

    clockType := py.NewType("Clock", "")
    clockType.Dict["tick"] = py.MustNewMethod("tick", clock_tick, 0, "")

    Clock_new := func (module py.Object, args py.Tuple) (py.Object, error) {
        return &TimeClock{
            lastTime: time.Now(),
            pyType: clockType,
        }, nil
    }

    return &py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame.time",
            Doc: "pygame module for monitoring time",
        },
        Methods: []*py.Method{
            py.MustNewMethod("Clock", Clock_new, 0, ""),
        },
    }
}

// hack: return the Surface constructor as one of the return values
func createDisplayModule(system GraphicsSystem) (*py.ModuleImpl, PythonMethod) {
    var screenSurface *RegularSurface
    regularSurfaceType := py.NewType("Surface", "")

    pygame_draw := func (self py.Object, args py.Tuple) (py.Object, error) {
        if screenSurface != nil {
            system.AddDrawCall(func (screen *ebiten.Image) {
                var options ebiten.DrawImageOptions
                screen.DrawImage(screenSurface.Image, &options)
            })
        }

        system.Flip()
        return py.None, nil
    }

    pygame_surface_blit := func (self py.Object, args py.Tuple) (py.Object, error) {
        var arg1, arg2 py.Object
        err := py.UnpackTuple(args, py.StringDict{}, "", 1, 2, &arg1, &arg2)
        if err != nil {
            return py.None, fmt.Errorf("Unable to unpack blit args %v", err)
        }

        var x1 int
        var y1 int
        arg2Rect, ok := arg2.(*Rect)
        if ok {
            x1, _ = arg2Rect.x1.GoInt()
            y1, _ = arg2Rect.y1.GoInt()
        } else {
            return py.None, fmt.Errorf("Expected rect for second argument but was %v", arg2.Type().Name)
        }

        fromSurface, ok := arg1.(*RegularSurface)
        if !ok {
            return py.None, fmt.Errorf("Expected surface for first argument but was %v", arg1.Type().Name)
        }

        selfSurface, ok := self.(*RegularSurface)
        if !ok {
            return py.None, fmt.Errorf("Expected self to be a surface but was %v", self.Type().Name)
        }
        var options ebiten.DrawImageOptions
        options.GeoM.Translate(float64(x1), float64(y1))

        selfSurface.Image.DrawImage(fromSurface.Image, &options)

        return py.None, nil
    }

    pygame_surface_fill := func (self py.Object, args py.Tuple) (py.Object, error) {
        var arg1 py.Object

        // log.Printf("Args are %v", args)

        err := py.UnpackTuple(args, py.StringDict{}, "", 1, 1, &arg1)
        if err != nil {
            return py.None, fmt.Errorf("Unable to unpack display args %v", err)
        }

        colorTuple, ok := arg1.(py.Tuple)

        if !ok {
            return py.None, fmt.Errorf("Expected tuple for color")
        }

        var r, g, b py.Int

        err = py.LoadTuple(colorTuple, []interface{}{&r, &g, &b})
        if err != nil {
            return py.None, fmt.Errorf("Unable to load fill args %v", err)
        }

        rInt, err := r.GoInt()
        if err != nil {
            return py.None, fmt.Errorf("Unable to convert r to int %v", err)
        }

        gInt, err := g.GoInt()
        if err != nil {
            return py.None, fmt.Errorf("Unable to convert g to int %v", err)
        }

        bInt, err := b.GoInt()
        if err != nil {
            return py.None, fmt.Errorf("Unable to convert b to int %v", err)
        }

        fillColor := color.RGBA{R: uint8(rInt), G: uint8(gInt), B: uint8(bInt), A: 255}

        surface, ok := self.(*RegularSurface)
        if !ok {
            return py.None, fmt.Errorf("Not a surface")
        }

        surface.Image.Fill(fillColor)

        return py.None, nil
    }

    RegularSurface_new := func (module py.Object, args py.Tuple) (py.Object, error) {
        var arg1 py.Object

        // log.Printf("Args are %v", args)

        err := py.UnpackTuple(args, py.StringDict{}, "", 1, 1, &arg1)
        if err != nil {
            return py.None, fmt.Errorf("Unable to unpack surface args %v", err)
        }

        sizeTuple, ok := arg1.(py.Tuple)

        if !ok {
            return py.None, fmt.Errorf("Expected tuple for size")
        }

        var width, height py.Int

        err = py.LoadTuple(sizeTuple, []interface{}{&width, &height})
        if err != nil {
            return py.None, fmt.Errorf("Unable to load surface args %v", err)
        }

        widthInt, err := width.GoInt()
        if err != nil {
            return py.None, fmt.Errorf("Unable to convert width to int %v", err)
        }

        heightInt, err := height.GoInt()
        if err != nil {
            return py.None, fmt.Errorf("Unable to convert height to int %v", err)
        }

        // log.Printf("Create new surface %d %d", widthInt, heightInt)

        surface := &RegularSurface{
            Image: ebiten.NewImage(widthInt, heightInt),
            pyType: regularSurfaceType,
        }
        return surface, nil
    }

    pygame_set_display_mode := func (self py.Object, args py.Tuple) (py.Object, error) {
        // log.Printf("set display mode")
        var size py.Object

        // log.Printf("Args are %v", args)

        err := py.UnpackTuple(args, py.StringDict{}, "", 1, 1, &size)
        if err != nil {
            return py.None, fmt.Errorf("Unable to unpack display args %v", err)
        }

        sizeTuple, ok := size.(py.Tuple)
        if ok {
            var width, height py.Int
            err = py.LoadTuple(sizeTuple, []interface{}{&width, &height})
            if err != nil {
                return py.None, err
            }

            widthInt, err := width.GoInt()
            if err != nil {
                return py.None, err
            }

            heightInt, err := height.GoInt()
            if err != nil {
                return py.None, err
            }

            system.SetScreenSize(widthInt, heightInt)

            surface := RegularSurface{
                Image: ebiten.NewImage(widthInt, heightInt),
                pyType: regularSurfaceType,
            }

            screenSurface = &surface

            return &surface, nil
        } else {
            return py.None, fmt.Errorf("Expected tuple for display size")
        }
    }

    regularSurfaceType.Dict["fill"] = py.MustNewMethod("fill", pygame_surface_fill, 0, "fill surface with color")
    regularSurfaceType.Dict["blit"] = py.MustNewMethod("blit", pygame_surface_blit, 0, "blit one surface onto another")

    return &py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame.display",
            Doc: "pygame module to control the display window and screen",
        },
        Methods: []*py.Method{
            py.MustNewMethod("set_mode", pygame_set_display_mode, 0, "set display mode"),
            py.MustNewMethod("update", pygame_draw, 0, "update display"),
        },
        Globals: py.StringDict{},
        OnContextClosed: func (instance *py.Module) {
        },
    }, RegularSurface_new
}

func createRectType() PythonMethod {
    pygame_rect_move_ip := func (self py.Object, args py.Tuple) (py.Object, error) {
        var x, y py.Int

        err := py.LoadTuple(args, []interface{}{&x, &y})
        if err != nil {
            return py.None, fmt.Errorf("Error with move_ip args: %v", err)
        }

        rect := self.(*Rect)
        rect.x1 += x
        rect.y1 += y
        rect.x2 += x
        rect.y2 += y

        return py.None, nil
    }

    rectType := py.NewType("Rect", "")
    rectType.Dict["move_ip"] = py.MustNewMethod("move_ip", pygame_rect_move_ip, 0, "move rect")

    Rect_new := func (module py.Object, args py.Tuple) (py.Object, error) {
        var coord1, coord2 py.Object
        err := py.UnpackTuple(args, py.StringDict{}, "", 2, 2, &coord1, &coord2)
        if err != nil {
            return py.None, fmt.Errorf("Unable to unpack rect args %v", err)
        }

        coord1Tuple, ok := coord1.(py.Tuple)
        if !ok {
            return py.None, fmt.Errorf("Expected tuple for coord1")
        }

        var x1, y1, x2, y2 py.Int
        err = py.LoadTuple(coord1Tuple, []interface{}{&x1, &y1})
        if err != nil {
            return py.None, fmt.Errorf("Unable to load rect args %v", err)
        }

        coord2Tuple, ok := coord2.(py.Tuple)
        if !ok {
            return py.None, fmt.Errorf("Expected tuple for coord2")
        }

        err = py.LoadTuple(coord2Tuple, []interface{}{&x2, &y2})
        if err != nil {
            return py.None, fmt.Errorf("Unable to load rect args %v", err)
        }

        return &Rect{
            x1: x1,
            y1: y1,
            x2: x2,
            y2: y2,
            pyType: rectType,
        }, nil
    }

    return Rect_new
}

type InternalEventType int

const (
    NoEvent InternalEventType = iota
    QuitEvent
    KeyDownEvent
)

type EventType struct {
    kind InternalEventType
    pyType *py.Type
}

func (eventType *EventType) M__eq__(other py.Object) (py.Object, error) {
    otherEventType, ok := other.(*EventType)
    if !ok {
        return py.None, fmt.Errorf("Expected event type")
    }

    return py.Bool(eventType.kind == otherEventType.kind), nil
}

func (eventType *EventType) Type() *py.Type {
    return eventType.pyType
}

type Event struct {
    eventType *EventType
    pyType *py.Type
    key ebiten.Key
}

func (event *Event) Type() *py.Type {
    return event.pyType
}

type EventTypes struct {
    Quit *EventType
    KeyDown *EventType
}

func createEventModule(eventTypes *EventTypes, eventTypeType *py.Type) *py.ModuleImpl {
    eventType := py.NewType("Event", "")

    // returns a list of events
    event_get := func (self py.Object, args py.Tuple) (py.Object, error) {
        keys := inpututil.AppendJustPressedKeys(nil)

        var events []py.Object
        for _, key := range keys {
            if key == ebiten.KeyEscape || key == ebiten.KeyCapsLock {
                events = append(events, &Event{
                    eventType: eventTypes.Quit,
                    pyType: eventType,
                })
            } else {
                events = append(events, &Event{
                    eventType: eventTypes.KeyDown,
                    pyType: eventType,
                    key: key,
                })
            }
        }

        return py.NewListFromItems(events), nil
    }

    eventType.Dict["type"] = &py.Property{
        Fget: func (self py.Object) (py.Object, error) {
            event := self.(*Event)

            return event.eventType, nil
        },
    }

    eventType.Dict["key"] = &py.Property{
        Fget: func (self py.Object) (py.Object, error) {
            event := self.(*Event)

            return py.Int(event.key), nil
        },
    }

    return &py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame.event",
            Doc: "pygame module for interacting with events and queues",
        },
        Methods: []*py.Method{},
        Globals: py.StringDict{
            "get": py.MustNewMethod("get", event_get, 0, ""),
        },
        OnContextClosed: func (instance *py.Module) {
        },
    }
}

func RegisterPygameModule(system GraphicsSystem, context py.Context) error {
    log.Printf("Register pygame")

    pygame_init := func (self py.Object, args py.Tuple) (py.Object, error) {
        successes := 1
        failures := 0
        return py.SequenceTuple(py.NewListFromItems([]py.Object{py.Int(successes), py.Int(failures)}))
    }

    moduleStore := py.NewModuleStore()

    displayDefintion, RegularSurface_new := createDisplayModule(system)
    displayModule, err := moduleStore.NewModule(context, displayDefintion)

    if err != nil {
        return err
    }

    timeModule, err := moduleStore.NewModule(context, createTimeModule())
    if err != nil {
        return err
    }

    eventType := py.NewType("EventType", "")

    eventTypes := &EventTypes{
        Quit: &EventType{
            kind: QuitEvent,
            pyType: eventType,
        },
        KeyDown: &EventType{
            kind: KeyDownEvent,
            pyType: eventType,
        },
    }

    eventModule, err := moduleStore.NewModule(context, createEventModule(eventTypes, eventType))
    if err != nil {
        return err
    }

    Rect_new := createRectType()

    pygameMethods := []*py.Method{
        py.MustNewMethod("init", pygame_init, 0, "init"),
        py.MustNewMethod("Surface", RegularSurface_new, 0, ""),
        py.MustNewMethod("Rect", Rect_new, 0, ""),
    }

    py.RegisterModule(&py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame",
            Doc: "pygame module",
        },
        Methods: pygameMethods,
        Globals: py.StringDict{
            "display": displayModule,
            "time": timeModule,
            "event": eventModule,
            "QUIT": eventTypes.Quit,
            "KEYDOWN": eventTypes.KeyDown,
            "K_w": py.Int(ebiten.KeyW),
            "K_a": py.Int(ebiten.KeyA),
            "K_s": py.Int(ebiten.KeyS),
            "K_d": py.Int(ebiten.KeyD),
            "K_UP": py.Int(ebiten.KeyUp),
            "K_DOWN": py.Int(ebiten.KeyDown),
            "K_LEFT": py.Int(ebiten.KeyLeft),
            "K_RIGHT": py.Int(ebiten.KeyRight),
        },
        OnContextClosed: func (instance *py.Module) {
        },
    })

    // py.RegisterModule(displayModule)

    return nil
}
