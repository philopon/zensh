package progress

type spin struct {
	phase int
}

func (s *spin) next() {
	if s.phase >= 9 {
		s.phase = 0
	} else {
		s.phase += 1
	}
}

func (s spin) String() string {
	return string([]rune{s.toRune()})
}

func (s spin) toRune() rune {
	switch s.phase {
	case 0:
		return '⠋'
	case 1:
		return '⠙'
	case 2:
		return '⠹'
	case 3:
		return '⠸'
	case 4:
		return '⠼'
	case 5:
		return '⠴'
	case 6:
		return '⠦'
	case 7:
		return '⠧'
	case 8:
		return '⠇'
	default:
		return '⠏'
	}
}
