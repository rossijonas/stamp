package manager

// Yay implements the Adapter interface for Arch Linux's yay (AUR helper).
type Yay struct {
	archHelper
}

// NewYay creates a new yay adapter. AUR builds must not run as root; yay
// escalates internally, so commands are not wrapped in sudo.
func NewYay() *Yay {
	return &Yay{
		archHelper: archHelper{
			exec: defaultExecutor,
			name: "yay",
			bin:  "yay",
			flags: []string{
				"--answerclean", "None",
				"--answerdiff", "None",
				"--answeredit", "None",
			},
			upgradeFlags: []string{"--answerupgrade", "None"},
			escalate:     false,
			strictQu:     false,
		},
	}
}

var _ Adapter = (*Yay)(nil)
