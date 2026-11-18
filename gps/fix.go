package gps

type Fix struct {
	RMC RMC
	GGA GGA
}

const maxHDOP = 5

func (f *Fix) Update(s Sentence) {
	switch value := s.(type) {
	case RMC:
		f.RMC = value
	case GGA:
		f.GGA = value
	}
}

func (f *Fix) Trusted() bool {
	if !f.RMC.Valid {
		return false
	}

	switch f.GGA.FixQuality {
	case FIX_VALID, DIFFERENTIAL_GPS_FIX, PPS_FIX, RTK_FIXED, RTK_FLOAT:
		// These are bueno
	default:
		return false
	}
	if f.GGA.HDOP >= maxHDOP {
		return false
	}

	return true
}
