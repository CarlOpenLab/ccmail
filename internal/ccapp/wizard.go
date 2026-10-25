package ccapp

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"ccmail/internal/providers"
)

// Wizard is the add-account flow: pick a provider, fill in the form, connect.
type Wizard struct {
	Step     int // 1 provider, 2 form, 3 connecting, 4 done
	Provider string

	Email    string
	Password string
	Name     string
	Color    string
	AuthUser string

	IMAPHost string
	IMAPPort string
	SMTPHost string
	SMTPPort string
	Advanced bool

	Busy    bool
	Err     string
	Warning string
	Added   bool
}

func newWizard() *Wizard {
	return &Wizard{Step: 1, Color: accountColors[0]}
}

// openWizard shows the add-account flow above everything else.
func (a *App) openWizard() {
	a.settings = false
	a.wizard = newWizard()
}

// wizardModal draws the wizard.
func (a *App) wizardModal(c *ui.Context, p palette) {
	w := a.wizard
	open := true
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(560).MaxHeight(640).Clip().Background(p.Popover).Radius(16).
			Border(1, p.Border).Shadow(0, 22, 56, -12, p.Shadow).Children(func() {
			ui.Row(c).Padding(16, 20).Gap(10).AlignItems(ui.Center).Children(func() {
				title := "选择邮件服务商"
				if w.Step == 2 {
					title = "登录邮箱"
				} else if w.Step >= 3 {
					title = "连接账号"
				}
				ui.Text(c, title).FontSize(15.5).Bold().TextColor(p.Text).Grow(1)
				a.iconButton(c, iconClose, "关闭", p, func() { a.wizard = nil })
			})
			ui.Divider(c)

			ui.Scroll(c).Grow(1).Padding(18, 20, 20, 20).Children(func() {
				switch w.Step {
				case 1:
					a.wizardProviders(c, p, w)
				case 2:
					a.wizardForm(c, p, w)
				case 3:
					a.wizardConnecting(c, p, w)
				case 4:
					a.wizardDone(c, p, w)
				}
			})
		})
	})
	if !open {
		a.wizard = nil
	}
}

func (a *App) wizardProviders(c *ui.Context, p palette, w *Wizard) {
	ui.Text(c, "选择你使用的邮箱服务，下一步填写授权码。").FontSize(12.5).TextColor(p.Muted).
		Margin(0, 0, 14, 0)
	ui.Row(c).Gap(10).Wrap().Children(func() {
		for _, pr := range providers.All() {
			pr := pr
			tile := ui.Column(c).Width(158).Padding(12, 10).Radius(12).Gap(8).Center().
				Background(p.Chip).Border(1, p.Border)
			tile.Cursor(ui.CursorPointer)
			tile.Children(func() {
				ui.Box(c).Size(38, 38).Radius(11).Center().Background(ui.Hex(pr.Color)).Children(func() {
					ui.Text(c, pr.Monogram).FontSize(16).Bold().TextColor(ui.Hex("#ffffff"))
				})
				ui.Text(c, pr.Name).FontSize(12).TextColor(p.Text).TextAlign(ui.Center)
			})
			tile.OnClick(func() {
				w.Provider = pr.ID
				w.Color = pr.Color
				w.Step = 2
			})
		}
	})
}

func (a *App) wizardForm(c *ui.Context, p palette, w *Wizard) {
	pr := providers.ByID(w.Provider)
	hint := ""
	if pr != nil {
		hint = pr.Name + " · " + pr.Hint
	}
	if hint != "" {
		ui.Text(c, hint).FontSize(12).TextColor(p.Faint).Margin(0, 0, 12, 0).LineHeight(1.6)
	}

	ui.Form(c, func() {
		ui.Field(c, "邮箱地址", func() {
			ui.TextInput(c, &w.Email).Placeholder("name@example.com")
		})
		ui.Field(c, "授权码", func() {
			ui.TextInput(c, &w.Password).Placeholder("IMAP / SMTP 授权码或应用专用密码")
		})
		ui.Field(c, "显示名称", func() {
			ui.TextInput(c, &w.Name).Placeholder("例如：工作邮箱")
		})
		ui.Field(c, "账号颜色", func() {
			ui.Row(c).Gap(6).Children(func() {
				for _, hex := range accountColors {
					hex := hex
					sw := ui.Button(c, "").Size(22, 22).Radius(11).Padding(0).Label("颜色 " + hex)
					sw.Background(ui.Hex(hex))
					if strings.EqualFold(hex, w.Color) {
						sw.Border(2, p.Text)
					}
					sw.OnClick(func() { w.Color = hex })
				}
			})
		})
	})

	if w.Err != "" {
		ui.Text(c, w.Err).FontSize(12).TextColor(p.Danger).Margin(10, 0, 0, 0).LineHeight(1.6)
	}

	ui.Row(c).Gap(8).Margin(6, 0, 0, 0).Children(func() {
		ui.Button(c, "高级设置").OnClick(func() { w.Advanced = !w.Advanced })
		ui.Spacer(c)
		ui.Button(c, "上一步").OnClick(func() { w.Step = 1; w.Err = "" })
		a.cta(c, p, "连接并添加").OnClick(func() { a.startConnect(w) })
	})

	if w.Advanced {
		ui.Divider(c).Margin(14, 0, 10, 0)
		ui.Form(c, func() {
			ui.Field(c, "IMAP 服务器", func() {
				ui.TextInput(c, &w.IMAPHost).Placeholder("留空使用服务商预设")
			})
			ui.Field(c, "IMAP 端口", func() {
				ui.TextInput(c, &w.IMAPPort).Placeholder("993")
			})
			ui.Field(c, "SMTP 服务器", func() {
				ui.TextInput(c, &w.SMTPHost).Placeholder("留空使用服务商预设")
			})
			ui.Field(c, "SMTP 端口", func() {
				ui.TextInput(c, &w.SMTPPort).Placeholder("465")
			})
			ui.Field(c, "登录用户名", func() {
				ui.TextInput(c, &w.AuthUser).Placeholder("默认与邮箱地址相同")
			})
		})
	}
}

func (a *App) wizardConnecting(c *ui.Context, p palette, w *Wizard) {
	ui.Column(c).Fill().Center().Gap(10).Padding(20).Children(func() {
		if w.Busy {
			ui.Spinner(c)
			ui.Text(c, "正在连接 IMAP 验证凭据…").FontSize(13).TextColor(p.Muted)
			ui.Text(c, "验证通过后会挂载账号并开始同步收件箱。").FontSize(12).TextColor(p.Faint)
			return
		}
		ui.Icon(c, iconJunk).FontSize(28).TextColor(p.Danger)
		ui.Text(c, "连接失败").FontSize(15).Bold().TextColor(p.Text)
		ui.Text(c, w.Err).FontSize(12).TextColor(p.Muted).TextAlign(ui.Center).MaxWidth(380)
		ui.Row(c).Gap(8).Margin(10, 0, 0, 0).Children(func() {
			ui.Button(c, "返回修改").OnClick(func() {
				w.Step = 2
				w.Busy = false
			})
		})
	})
}

func (a *App) wizardDone(c *ui.Context, p palette, w *Wizard) {
	ui.Column(c).Fill().Center().Gap(8).Padding(20).Children(func() {
		ui.Box(c).Size(56, 56).Radius(28).Center().Background(p.Success.Alpha(0.16)).Children(func() {
			ui.Icon(c, iconCheck).FontSize(26).TextColor(p.Success)
		})
		ui.Text(c, "账号已添加").FontSize(15).Bold().TextColor(p.Text)
		if w.Warning != "" {
			ui.Text(c, w.Warning).FontSize(12).TextColor(p.Warning).TextAlign(ui.Center).MaxWidth(420)
		}
		ui.Text(c, "收件箱正在后台同步，新邮件会实时出现。").FontSize(12.5).TextColor(p.Faint)
		ui.Row(c).Gap(8).Margin(10, 0, 0, 0).Children(func() {
			a.cta(c, p, "打开收件箱").OnClick(func() {
				a.wizard = nil
				a.navigate(ViewSel{Kind: "unified", Name: "全部收件箱"})
			})
			ui.Button(c, "关闭").OnClick(func() { a.wizard = nil })
		})
	})
}

// startConnect verifies the account off the main thread.
func (a *App) startConnect(w *Wizard) {
	if w.Busy {
		return
	}
	if !strings.Contains(w.Email, "@") {
		w.Err = "请填写有效的邮箱地址"
		return
	}
	if w.Password == "" {
		w.Err = "请填写授权码"
		return
	}
	w.Err = ""
	w.Busy = true
	w.Step = 3

	input := AddAccountInput{
		Provider: w.Provider,
		Email:    w.Email,
		Password: w.Password,
		Name:     w.Name,
		Color:    w.Color,
		AuthUser: w.AuthUser,
		IMAPHost: w.IMAPHost,
		SMTPHost: w.SMTPHost,
	}
	if n := parseInt(w.IMAPPort); n > 0 {
		input.IMAPPort = n
	}
	if n := parseInt(w.SMTPPort); n > 0 {
		input.SMTPPort = n
	}

	go func() {
		warning, err := a.addAccount(input)
		a.update(func() {
			w.Busy = false
			if err != nil {
				w.Err = err.Error()
				return
			}
			w.Warning = warning
			w.Added = true
			w.Step = 4
		})
	}()
}

func parseInt(s string) int {
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
