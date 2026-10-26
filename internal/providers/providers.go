// Package providers holds presets for common email services.
package providers

// Server describes one protocol endpoint of a provider preset.
type Server struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Secure     bool   `json:"secure"`
	RequireTLS bool   `json:"requireTLS,omitempty"`
}

// Provider is a mail service preset shown in the add-account wizard.
type Provider struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Short    string   `json:"shortName"`
	Domains  []string `json:"domains"`
	Color    string   `json:"color"`
	Monogram string   `json:"monogram"`
	IMAP     Server   `json:"imap"`
	SMTP     Server   `json:"smtp"`
	Hint     string   `json:"hint"`
	// HelpURL is the provider's official page explaining how to enable
	// IMAP/SMTP and generate the auth code or app password. Empty for
	// custom servers, which have no official guide.
	HelpURL string `json:"helpUrl,omitempty"`
}

// All returns every built-in provider preset.
func All() []Provider { return all }

// ByID finds a provider preset by id.
func ByID(id string) *Provider {
	for i := range all {
		if all[i].ID == id {
			return &all[i]
		}
	}
	return nil
}

// Guess returns the preset matching an email domain, or nil.
func Guess(email string) *Provider {
	for i := 0; i < len(email); i++ {
		if email[i] != '@' {
			continue
		}
		domain := email[i+1:]
		for j := range all {
			for _, d := range all[j].Domains {
				if d == domain {
					return &all[j]
				}
			}
		}
		return nil
	}
	return nil
}

var all = []Provider{
	{
		ID: "qq", Name: "QQ 邮箱", Short: "QQ", Domains: []string{"qq.com", "foxmail.com"},
		Color: "#12b7f5", Monogram: "Q",
		IMAP:    Server{Host: "imap.qq.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.qq.com", Port: 465, Secure: true},
		Hint:    "需要「授权码」而不是 QQ 密码：QQ 邮箱网页版 → 设置 → 账户 → 开启 IMAP/SMTP 服务 → 生成授权码。",
		HelpURL: "https://help.mail.qq.com/detail/0/1087",
	},
	{
		ID: "gmail", Name: "Gmail", Short: "Gmail", Domains: []string{"gmail.com", "googlemail.com"},
		Color: "#ea4335", Monogram: "G",
		IMAP:    Server{Host: "imap.gmail.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.gmail.com", Port: 465, Secure: true},
		Hint:    "Google 账号需开启两步验证后生成「应用专用密码」（App Password），用 16 位应用密码登录。",
		HelpURL: "https://support.google.com/accounts/answer/185833",
	},
	{
		ID: "163", Name: "网易 163", Short: "163", Domains: []string{"163.com"},
		Color: "#e0382c", Monogram: "1",
		IMAP:    Server{Host: "imap.163.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.163.com", Port: 465, Secure: true},
		Hint:    "需要在 163 邮箱设置中开启 IMAP/SMTP 并使用「客户端授权密码」。",
		HelpURL: "https://help.mail.163.com/faqDetail.do?code=d7a5dc8471cd0c0e8b4b8f4f8e49998b374173cfe9171305fa1ce630d7f67ac2a5feb28b66796d3b",
	},
	{
		ID: "126", Name: "网易 126", Short: "126", Domains: []string{"126.com"},
		Color: "#f5a623", Monogram: "2",
		IMAP:    Server{Host: "imap.126.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.126.com", Port: 465, Secure: true},
		Hint:    "需要在 126 邮箱设置中开启 IMAP/SMTP 并使用「客户端授权密码」。",
		HelpURL: "https://help.mail.126.com/faqDetail.do?code=d7a5dc8471cd0c0e8b4b8f4f8e49998b374173cfe9171305fa1ce630d7f67ac286624f309a1a7089",
	},
	{
		ID: "outlook", Name: "Outlook / Microsoft 365", Short: "Outlook", Domains: []string{"outlook.com", "hotmail.com", "live.com", "msn.com"},
		Color: "#0f6cbd", Monogram: "O",
		IMAP:    Server{Host: "outlook.office365.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.office365.com", Port: 587, Secure: false, RequireTLS: true},
		Hint:    "个人账户需在 Microsoft 账号安全中心开启两步验证并创建应用密码。",
		HelpURL: "https://support.microsoft.com/en-us/account-billing/how-to-get-and-use-app-passwords-5896ed9b-4263-e681-128a-a6f2979a7944",
	},
	{
		ID: "icloud", Name: "iCloud 邮箱", Short: "iCloud", Domains: []string{"icloud.com", "me.com", "mac.com"},
		Color: "#3693f3", Monogram: "i",
		IMAP:    Server{Host: "imap.mail.me.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.mail.me.com", Port: 587, Secure: false, RequireTLS: true},
		Hint:    "需要 Apple ID 账户下的「App 专用密码」（appleid.apple.com → 登录与安全）。",
		HelpURL: "https://support.apple.com/en-us/102654",
	},
	{
		ID: "exmail", Name: "企业微信邮箱", Short: "企业邮",
		Color: "#0082ef", Monogram: "企",
		IMAP:    Server{Host: "imap.exmail.qq.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.exmail.qq.com", Port: 465, Secure: true},
		Hint:    "使用完整邮箱地址与邮箱密码（或安全登录的客户端专用密码）。",
		HelpURL: "https://open.work.weixin.qq.com/help2/pc/19902",
	},
	{
		ID: "aliyun", Name: "阿里企业邮箱", Short: "阿里邮",
		Color: "#ff6a00", Monogram: "阿",
		IMAP:    Server{Host: "imap.mxhichina.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.qiye.aliyun.com", Port: 465, Secure: true},
		Hint:    "使用完整邮箱地址与企业邮箱密码。",
		HelpURL: "https://help.aliyun.com/zh/document_detail/444269.html",
	},
	{
		ID: "yahoo", Name: "Yahoo Mail", Short: "Yahoo", Domains: []string{"yahoo.com"},
		Color: "#6001d2", Monogram: "Y",
		IMAP:    Server{Host: "imap.mail.yahoo.com", Port: 993, Secure: true},
		SMTP:    Server{Host: "smtp.mail.yahoo.com", Port: 465, Secure: true},
		Hint:    "需要在 Yahoo 账号安全设置中生成「应用密码」。",
		HelpURL: "https://help.yahoo.com/kb/SLN15241.html",
	},
	{
		ID: "custom", Name: "自定义 IMAP/SMTP", Short: "自定义",
		Color: "#6e7b8b", Monogram: "@",
		IMAP: Server{Host: "", Port: 993, Secure: true},
		SMTP: Server{Host: "", Port: 465, Secure: true},
		Hint: "填写你邮件服务商提供的 IMAP / SMTP 服务器地址。",
	},
}
