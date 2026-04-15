package crawler

// progressReporter can be embedded in a runner to provide a progress callback.
// Runners that embed it automatically satisfy Runner.SetProgress.
type progressReporter struct {
	progressFn func(int)
}

func (p *progressReporter) SetProgress(fn func(visited int)) {
	p.progressFn = fn
}

func (p *progressReporter) reportProgress(visited int) {
	if p.progressFn != nil {
		p.progressFn(visited)
	}
}
