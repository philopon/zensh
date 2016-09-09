package progress

import ansi "github.com/k0kubun/go-ansi"

// Term Manager
type TermManager struct {
	y    int
	maxY int
}

func (tm *TermManager) MoveY(y int) {
	dy := y - tm.y
	if dy < 0 {
		ansi.CursorUp(-dy)
	} else if dy > 0 {
		if y > tm.maxY {
			btm := tm.maxY - tm.y
			if btm < 0 {
				btm = 0
			} else {
				ansi.CursorDown(btm)
			}

			for i := 0; i < y-btm; i++ {
				ansi.Println()
			}
		} else {
			ansi.CursorDown(dy)
		}
	}
	tm.y = y
}

func (tm *TermManager) Move(x, y int) {
	tm.MoveY(y)
	ansi.CursorHorizontalAbsolute(x + 1)
}

func (tm *TermManager) Write(s string) error {
	_, err := ansi.Print(s)
	return err
}

func (tm *TermManager) Writeln(s string) error {
	_, err := ansi.Println(s)
	tm.y += 1
	if tm.y > tm.maxY {
		tm.maxY = tm.y
	}
	return err
}

func (tm *TermManager) EraceRight() {
	ansi.EraseInLine(0)
}

func (tm *TermManager) Erace() {
	ansi.EraseInLine(2)
}

func (tm *TermManager) ToEnd() {
	tm.Move(0, tm.maxY)
}
