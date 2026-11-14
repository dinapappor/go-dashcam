package gps

type Fix struct {
	RMC RMC
	GGA GGA
}

func (f *Fix) Update(s Sentence) {
	switch value := s.(type) {
	case RMC:
		f.RMC = value
	case GGA:
		f.GGA = value
	}
}
