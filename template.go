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
<body style="margin:0;padding:0;background-color:#09090B;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#FAFAFA">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#09090B">
    <tr>
      <td align="center" style="padding:48px 24px">
        <table role="presentation" cellpadding="0" cellspacing="0" style="max-width:460px;width:100%">
          <tr>
            <td align="center" style="padding-bottom:36px">
              <span style="font-size:24px;font-weight:700;letter-spacing:-0.3px">{{.AppName}}</span>
            </td>
          </tr>
          <tr>
            <td align="center" style="padding-bottom:12px">
              <p style="margin:0;font-size:16px;line-height:24px;color:#A1A1AA">Your sign-in code</p>
            </td>
          </tr>
          <tr>
            <td align="center" style="padding-bottom:24px">
              <div style="display:inline-block;padding:14px 18px;border-radius:10px;background:#18181B;border:1px solid #27272A;font-family:'SF Mono',SFMono-Regular,Menlo,Consolas,monospace;font-size:28px;letter-spacing:6px;color:#FAFAFA">{{.Code}}</div>
            </td>
          </tr>
          <tr>
            <td align="center" style="padding-bottom:28px">
              <a href="{{.MagicURL}}" style="display:inline-block;background:#FAFAFA;color:#09090B;text-decoration:none;font-size:15px;font-weight:600;padding:14px 34px;border-radius:10px">
                Sign in to {{.AppName}}
              </a>
            </td>
          </tr>
          <tr>
            <td align="center" style="color:#52525B;font-size:13px;line-height:20px">
              This code expires soon. If you did not request it, you can ignore this email.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`))

// Render returns a subject and HTML body.
func (DefaultEmailRenderer) Render(code, magicURL, appName string) (subject, htmlBody string) {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "Your App"
	}

	var buf bytes.Buffer
	data := struct {
		AppName  string
		Code     string
		MagicURL string
	}{
		AppName:  appName,
		Code:     code,
		MagicURL: magicURL,
	}

	if err := defaultEmailTemplate.Execute(&buf, data); err != nil {
		// Template execution is deterministic and pre-validated. Keep a fallback to avoid hard failure.
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
