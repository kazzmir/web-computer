package py

import (
    "log"
)

type Quitter struct{
    Code int
}

var (
	QuitType = NewType("Quitter", "quit(code=0) -> <void>\n\nExits the program.")
    
	// Some well known bools
    Quit = Quitter{Code: 0}
)

// Type of this object
func (quit Quitter) Type() *Type {
	return QuitType
}

func (quit Quitter) M__call__(args Tuple, kwargs StringDict) (Object, error) {
    log.Printf("Quit python vm")
    return None, nil
}

