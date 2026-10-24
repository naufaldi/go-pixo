package png

// workProgress reports completed units, never elapsed-time estimates. Pixel loops
// check at most once per chunk; callbacks are bounded to 101 per phase.
type workProgress struct {
	callback                func(string, int)
	phase                   string
	total, step, next, last int
}

func newWorkProgress(callback func(string, int), phase string, total, minStep int) workProgress {
	step := max(minStep, (total+99)/100, 1)
	p := workProgress{callback: callback, phase: phase, total: total, step: step, next: step}
	if callback == nil {
		p.next = total + 1
	} else {
		callback(phase, 0)
	}
	return p
}

func (p *workProgress) report(completed int) {
	if p.callback == nil || completed < p.next || p.total <= 0 {
		return
	}
	percent := min(100, completed*100/p.total)
	if percent > p.last {
		p.callback(p.phase, percent)
		p.last = percent
	}
	p.next = completed + p.step
}

func (p *workProgress) finish() {
	if p.callback != nil && p.last != 100 {
		p.callback(p.phase, 100)
		p.last = 100
	}
}
