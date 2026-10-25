package ccapp

// rowVM is one line of the message list: either a date heading or a message.
type rowVM struct {
	header string  // set on heading rows
	msg    *MsgRef // nil on heading rows
}

func (r rowVM) isHeader() bool { return r.msg == nil }

// rebuildRows groups the messages under date headings.
func (a *App) rebuildRows() {
	rows := make([]rowVM, 0, len(a.msgs)+6)
	last := ""
	for _, m := range a.msgs {
		if label := dateGroupLabel(m.Env.Date); label != last {
			last = label
			rows = append(rows, rowVM{header: label})
		}
		rows = append(rows, rowVM{msg: m})
	}
	a.rows = rows
}

// rowOf maps a list row back to its message.
func (a *App) rowOf(i int) *MsgRef {
	if i < 0 || i >= len(a.rows) {
		return nil
	}
	return a.rows[i].msg
}

// rowKey gives each row a stable identity for the native list.
func (a *App) rowKey(i int) any {
	if i < 0 || i >= len(a.rows) {
		return i
	}
	r := a.rows[i]
	if r.isHeader() {
		return "h:" + r.header
	}
	return r.msg.Key()
}

func (a *App) rowLabel(i int) string {
	if i < 0 || i >= len(a.rows) {
		return ""
	}
	r := a.rows[i]
	if r.isHeader() {
		return r.header
	}
	return displayName(r.msg.Env.From) + "，" + r.msg.Env.Subject
}

// toastMsg is a queued toast, shown by the next frame.
type toastMsg struct {
	text string
	kind string // "", "success", "error"
}

// flash queues a toast for the view to show.
func (a *App) flash(text, kind string) {
	a.toasts = append(a.toasts, toastMsg{text: text, kind: kind})
}

// AddAccountInput is what the wizard collects for a new mailbox.
type AddAccountInput struct {
	Provider string
	Email    string
	Password string
	Name     string
	Color    string
	AuthUser string
	IMAPHost string
	IMAPPort int
	SMTPHost string
	SMTPPort int
}
