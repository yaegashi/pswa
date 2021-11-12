package config

type NavigationFallback struct {
	Rewrite  string    `json:"rewrite"`
	Exclude  []string  `json:"exclude"`
	Globbers []Globber `json:"-"`
}

func (f *NavigationFallback) Compile() error {
	f.Globbers = make([]Globber, len(f.Exclude))
	for i := 0; i < len(f.Exclude); i++ {
		err := f.Globbers[i].Compile(f.Exclude[i])
		if err != nil {
			return err
		}
	}
	return nil
}
