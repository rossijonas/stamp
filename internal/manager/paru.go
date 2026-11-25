package manager

// Paru implements the Adapter interface for Arch Linux's Paru (AUR helper).
type Paru struct {
	archHelper
}

// NewParu creates a new Paru adapter.
func NewParu() *Paru {
	return &Paru{
		archHelper: archHelper{
			exec:     defaultExecutor,
			name:     "paru",
			bin:      "paru",
			escalate: false,
			strictQu: false,
		},
	}
}

var _ Adapter = (*Paru)(nil)
