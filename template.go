package magiclink

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"strings"
)

// DefaultEmailRenderer renders a default dark-themed magic-link email.
type DefaultEmailRenderer struct{}

var defaultEmailTemplate = template.Must(template.New("magiclink-email").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Sign in to {{.AppName}}</title>
</head>
<body style="margin:0;padding:0;background-color:#09090B;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#09090B">
<tr><td align="center" style="padding:48px 24px">
<table role="presentation" cellpadding="0" cellspacing="0" style="max-width:440px;width:100%">

<tr><td align="center" style="padding-bottom:40px">
  <span style="font-size:24px;font-weight:700;color:#FAFAFA;letter-spacing:-0.5px">{{.AppNameLower}}</span>
</td></tr>

<tr><td align="center" style="padding-bottom:28px">
  <p style="margin:0;font-size:16px;line-height:24px;color:#A1A1AA">Your sign-in code</p>
</td></tr>

<tr><td align="center" style="padding-bottom:32px">
  <table role="presentation" cellpadding="0" cellspacing="0"
    style="border-radius:12px;border:1px solid #27272A;background:#18181B;padding:16px 20px">
  <tr>{{.DigitCells}}</tr>
  </table>
</td></tr>

<tr><td align="center" style="padding-bottom:28px">
  <table role="presentation" cellpadding="0" cellspacing="0" style="width:280px;margin:0 auto">
  <tr>
    <td width="45%" style="border-bottom:1px solid #27272A;line-height:1px;font-size:1px">&nbsp;</td>
    <td width="10%" align="center" style="padding:0 14px;color:#52525B;font-size:12px;white-space:nowrap">or</td>
    <td width="45%" style="border-bottom:1px solid #27272A;line-height:1px;font-size:1px">&nbsp;</td>
  </tr></table>
</td></tr>

<tr><td align="center" style="padding-bottom:44px">
  <a href="{{.MagicURL}}" style="display:inline-block;background:#FAFAFA;color:#09090B;text-decoration:none;
    font-size:15px;font-weight:600;padding:14px 36px;border-radius:10px">Sign in to {{.AppName}}</a>
</td></tr>

<tr><td align="center">
  <p style="margin:0 0 8px;font-size:13px;line-height:20px;color:#52525B">
    This code expires in 10 minutes. If you didn't request this, you can safely ignore it.</p>
  <p style="margin:0;font-size:12px;color:#3F3F46">{{.AppName}} &mdash; {{.Tagline}}</p>
</td></tr>

</table>
</td></tr></table>
</body>
</html>`))

// Render returns a subject and HTML body.
func (DefaultEmailRenderer) Render(code, magicURL, appName string) (subject, htmlBody string) {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "Your App"
	}

	digitStyle := `style="font-family:'SF Mono',SFMono-Regular,Menlo,Consolas,monospace;font-size:32px;font-weight:700;color:#FAFAFA;width:44px;height:56px;text-align:center;vertical-align:middle"`
	var digits strings.Builder
	for i, ch := range code {
		if len(code) == 6 && i == 3 {
			digits.WriteString(`<td width="20"></td>`)
		}
		fmt.Fprintf(&digits, `<td %s>%s</td>`, digitStyle, string(ch))
	}

	var buf bytes.Buffer
	data := struct {
		AppName      string
		AppNameLower string
		Code         string
		MagicURL     string
		DigitCells   template.HTML
		Tagline      string
	}{
		AppName:      appName,
		AppNameLower: strings.ToLower(appName),
		Code:         code,
		MagicURL:     magicURL,
		DigitCells:   template.HTML(digits.String()),
		Tagline:      "powered by magic",
	}

	if err := defaultEmailTemplate.Execute(&buf, data); err != nil {
		htmlBody = "<p>Your sign-in code: " + code + "</p><p><a href=\"" + magicURL + "\">Sign in</a></p>"
	} else {
		htmlBody = buf.String()
	}

	subject = fmt.Sprintf("%s — your %s sign-in code", code, appName)
	return subject, htmlBody
}

var verifySuccessTemplate = template.Must(template.New("verify-success").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.AppName}} — Signed In</title>
  <style>
    body { font-family: -apple-system, system-ui, sans-serif; background: #09090B; color: #FAFAFA; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; }
    .card { text-align: center; padding: 2rem; }
    h1 { font-size: 2rem; letter-spacing: -1px; margin-bottom: .5rem; }
    p { color: #A1A1AA; margin-bottom: 1rem; }
    .check { font-size: 3rem; margin-bottom: 1rem; }
    a { color: #FAFAFA; }
  </style>
</head>
<body>
  <div class="card">
    <div class="check">&#10003;</div>
    <h1>You're in</h1>
    <p>Return to the {{.AppName}} app.</p>
    {{if .RedirectURL}}<p><a href="{{.RedirectLink}}">Continue</a></p>{{end}}
  </div>
  {{if .RedirectURL}}
  <script>
    try { window.location.href = "{{.RedirectURL}}"; } catch (e) {}
  </script>
  {{end}}
</body>
</html>`))

func renderVerifySuccessPage(appName, deepLinkURL string, result *AuthResult) (string, error) {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "Your App"
	}

	redirectURL := ""
	if strings.TrimSpace(deepLinkURL) != "" {
		u, err := url.Parse(deepLinkURL)
		if err != nil {
			return "", fmt.Errorf("invalid deep link url: %w", err)
		}
		q := u.Query()
		q.Set("token", result.JWT)
		q.Set("user_id", result.UserID)
		u.RawQuery = q.Encode()
		redirectURL = u.String()
	}

	var buf bytes.Buffer
	data := struct {
		AppName      string
		RedirectURL  string
		RedirectLink template.URL
	}{
		AppName:      appName,
		RedirectURL:  redirectURL,
		RedirectLink: template.URL(redirectURL),
	}
	if err := verifySuccessTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render verify success page: %w", err)
	}
	return buf.String(), nil
}
