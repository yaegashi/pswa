package config

type NavigationFallback struct {
	Rewrite  string    `json:"rewrite"`
	Exclude  []string  `json:"exclude"`
	Globbers []Globber `json:"-"`
}

func (f *NavigationFallback) Compile() error {
	f.Globbers = make([]Globber, len(f.Exclude)+1)
	err := f.Globbers[0].Compile("/.auth/*")
	if err != nil {
		return err
	}
	for i := 0; i < len(f.Exclude); i++ {
		err := f.Globbers[i+1].Compile(f.Exclude[i])
		if err != nil {
			return err
		}
	}
	return nil
}
