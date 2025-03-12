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

func RegisterPygameModule(system GraphicsSystem, context py.Context) error {
    pygame_draw := func (self py.Object, args py.Tuple) (py.Object, error) {
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

    methods := []*py.Method{
        py.MustNewMethod("draw", pygame_draw, 0, ""),
        py.MustNewMethod("clear", pygame_clear, 0, "clear screen to a color"),
        py.MustNewMethod("init", pygame_init, 0, "init"),
    }

    surfaceType := py.NewType("Surface", "")
    _ = surfaceType

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

            system.SetScreenSize(int(width), int(height))

            surface := DisplaySurface{
                pyType: surfaceType,
            }

            return &surface, nil
        } else {
            return py.None, fmt.Errorf("Expected tuple for display size")
        }

        /*
        err := py.LoadTuple(args, []interface{}{&size})
        if err != nil {
            return py.None, fmt.Errorf("Unable to load display args %v", err)
        }

        return py.None, nil
        */
    }

    moduleStore := py.NewModuleStore()

    displayModule, err := moduleStore.NewModule(context, &py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame.display",
            Doc: "pygame display",
        },
        Methods: []*py.Method{
            py.MustNewMethod("set_mode", pygame_set_display_mode, 0, "set display mode"),
        },
        Globals: py.StringDict{},
        OnContextClosed: func (instance *py.Module) {
        },
    })

    if err != nil {
        return err
    }

    py.RegisterModule(&py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame",
            Doc: "pygame module",
        },
        Methods: methods,
        Globals: py.StringDict{
            "display": displayModule,
        },
        OnContextClosed: func (instance *py.Module) {
        },
    })

    // py.RegisterModule(displayModule)

    return nil
}
