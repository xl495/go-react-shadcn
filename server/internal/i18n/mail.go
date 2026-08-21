package i18n

import "fmt"

func ResetPasswordMail(locale, name, link string) (subject, body string) {
	if Normalize(locale) == ZhCN {
		return "重置 gra 密码", fmt.Sprintf("您好 %s，\n\n请在 30 分钟内点击下面的链接重置密码：\n%s\n\n若非本人操作，请忽略此邮件。\n", name, link)
	}
	return "Reset your gra password", fmt.Sprintf("Hi %s,\n\nUse this link within 30 minutes to reset your password:\n%s\n\nIf you did not ask for this, you can ignore the email.\n", name, link)
}

func VerifyEmailMail(locale, link string) (subject, body string) {
	if Normalize(locale) == ZhCN {
		return "验证 gra 账号", "点击链接完成邮箱验证：" + link
	}
	return "Verify your gra account", "Click to verify your email: " + link
}
