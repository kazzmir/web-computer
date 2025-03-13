package pygame

import (
    "log"
    "fmt"
    "image/color"
    // "reflect"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/go-python/gpython/py"
)

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

func RegisterPygameModule(system GraphicsSystem, context py.Context) error {
    var screenSurface *RegularSurface

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

    pygame_clear := func (self py.Object, args py.Tuple) (py.Object, error) {
        var value py.Int
        err := py.LoadTuple(args, []interface{}{&value})
        if err != nil {
            log.Printf("Unable to load args: %v", err)
            return py.None, err
        }

        num, err := value.GoInt()
        if err != nil {
            log.Printf("Unable to convert to int: %v", err)
            return py.None, err
        }

        system.AddDrawCall(func (screen *ebiten.Image) {
            var c color.Color
            switch num {
                case 0: c = color.RGBA{R: 255, G: 0, B: 0, A: 255}
                case 1: c = color.RGBA{R: 0, G: 255, B: 0, A: 255}
                case 2: c = color.RGBA{R: 0, G: 0, B: 255, A: 255}
                case 3: c = color.RGBA{R: 255, G: 255, B: 0, A: 255}
                default: c = color.RGBA{R: 0, G: 0, B: 0, A: 255}
            }

            screen.Fill(c)
        })

        return py.None, nil
    }

    pygame_init := func (self py.Object, args py.Tuple) (py.Object, error) {
        successes := 1
        failures := 0
        return py.SequenceTuple(py.NewListFromItems([]py.Object{py.Int(successes), py.Int(failures)}))
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

    /*
    surfaceType := py.NewType("Surface", "")
    surfaceType.Dict["fill"] = py.MustNewMethod("fill", pygame_surface_fill, 0, "fill surface with color")
    */

    regularSurfaceType := py.NewType("Surface", "")
    regularSurfaceType.Dict["fill"] = py.MustNewMethod("fill", pygame_surface_fill, 0, "fill surface with color")
    regularSurfaceType.Dict["blit"] = py.MustNewMethod("blit", pygame_surface_blit, 0, "blit one surface onto another")

    rectType := py.NewType("Rect", "")

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

    moduleStore := py.NewModuleStore()

    displayModule, err := moduleStore.NewModule(context, &py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame.display",
            Doc: "pygame display",
        },
        Methods: []*py.Method{
            py.MustNewMethod("set_mode", pygame_set_display_mode, 0, "set display mode"),
            py.MustNewMethod("update", pygame_draw, 0, "update display"),
        },
        Globals: py.StringDict{},
        OnContextClosed: func (instance *py.Module) {
        },
    })

    if err != nil {
        return err
    }

    pygameMethods := []*py.Method{
        py.MustNewMethod("draw", pygame_draw, 0, ""),
        py.MustNewMethod("clear", pygame_clear, 0, "clear screen to a color"),
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
        },
        OnContextClosed: func (instance *py.Module) {
        },
    })

    // py.RegisterModule(displayModule)

    return nil
}
