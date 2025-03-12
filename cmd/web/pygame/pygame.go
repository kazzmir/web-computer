package pygame

import (
    "log"
    "image/color"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/go-python/gpython/py"
)

type GraphicsSystem interface {
    Flip()
    AddDrawCall(func(*ebiten.Image))
}

func RegisterPygameModule(system GraphicsSystem) {
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

    methods := []*py.Method{
        py.MustNewMethod("draw", pygame_draw, 0, ""),
        py.MustNewMethod("clear", pygame_clear, 0, "clear screen to a color"),
    }

    py.RegisterModule(&py.ModuleImpl{
        Info: py.ModuleInfo{
            Name: "pygame",
            Doc: "pygame module",
        },
        Methods: methods,
        Globals: py.StringDict{},
        OnContextClosed: func (instance *py.Module) {
        },
    })
}
