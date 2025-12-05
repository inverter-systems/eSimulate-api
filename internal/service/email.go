package service

import (
	"fmt"
	"html/template"
	"net/smtp"
	"os"
)

type EmailService struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

func NewEmailService() *EmailService {
	return &EmailService{
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("SMTP_FROM_EMAIL", "noreply@esimulate.com"),
		FromName:     getEnv("SMTP_FROM_NAME", "eSimulate"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// SendEmail envia um email simples
func (e *EmailService) SendEmail(to, subject, body string) error {
	// Se não houver configuração SMTP, apenas logar (modo desenvolvimento)
	if e.SMTPUser == "" || e.SMTPPassword == "" {
		fmt.Printf("[EMAIL SIMULADO] Enviado para: %s | Assunto: %s\n", to, subject)
		return nil
	}

	// Configurar autenticação
	auth := smtp.PlainAuth("", e.SMTPUser, e.SMTPPassword, e.SMTPHost)

	// Montar mensagem
	msg := []byte(fmt.Sprintf("From: %s <%s>\r\n", e.FromName, e.FromEmail) +
		fmt.Sprintf("To: %s\r\n", to) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	// Enviar email
	addr := fmt.Sprintf("%s:%s", e.SMTPHost, e.SMTPPort)
	return smtp.SendMail(addr, auth, e.FromEmail, []string{to}, msg)
}

// SendVerificationEmail envia email de verificação
func (e *EmailService) SendVerificationEmail(to, name, token string) error {
	verifyURL := fmt.Sprintf("%s/#/verify-email?token=%s", getEnv("APP_URL", "http://localhost:3000"), token)
	
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>Bem-vindo ao eSimulate!</h2>
			<p>Olá %s,</p>
			<p>Obrigado por se cadastrar. Clique no link abaixo para verificar seu email:</p>
			<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verificar Email</a></p>
			<p>Ou copie e cole este link no navegador:</p>
			<p>%s</p>
			<p>Se você não se cadastrou, ignore este email.</p>
			<hr>
			<p style="color: #666; font-size: 12px;">Powered by eSimulate</p>
		</body>
		</html>
	`, name, verifyURL, verifyURL)
	
	return e.SendEmail(to, "Verifique seu email - eSimulate", body)
}

// SendPasswordResetEmail envia email de recuperação de senha
func (e *EmailService) SendPasswordResetEmail(to, name, token string) error {
	resetURL := fmt.Sprintf("%s/#/reset-password?token=%s", getEnv("APP_URL", "http://localhost:3000"), token)
	
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>Recuperação de Senha</h2>
			<p>Olá %s,</p>
			<p>Recebemos uma solicitação para redefinir sua senha. Clique no link abaixo:</p>
			<p><a href="%s" style="background-color: #2196F3; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Redefinir Senha</a></p>
			<p>Ou copie e cole este link no navegador:</p>
			<p>%s</p>
			<p>Este link expira em 1 hora. Se você não solicitou esta alteração, ignore este email.</p>
			<hr>
			<p style="color: #666; font-size: 12px;">Powered by eSimulate</p>
		</body>
		</html>
	`, name, resetURL, resetURL)
	
	return e.SendEmail(to, "Recuperação de Senha - eSimulate", body)
}

// SendCompanyInviteEmail envia convite da empresa para candidato
func (e *EmailService) SendCompanyInviteEmail(to, candidateName, companyName, companyLogo, linkToken string) error {
	examURL := fmt.Sprintf("%s/#/eval/%s", getEnv("APP_URL", "http://localhost:3000"), linkToken)
	
	logoHTML := ""
	if companyLogo != "" {
		logoHTML = fmt.Sprintf(`<img src="%s" alt="%s" style="max-width: 200px; margin: 20px 0;">`, companyLogo, companyName)
	}
	
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<div style="text-align: center;">
				%s
				<h2>%s</h2>
			</div>
			<h2>Olá %s!</h2>
			<p>A empresa <strong>%s</strong> convidou você para realizar um teste técnico.</p>
			<p style="text-align: center; margin: 30px 0;">
				<a href="%s" style="background-color: #4CAF50; color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-size: 16px; display: inline-block;">Iniciar Teste</a>
			</p>
			<p>Ou copie e cole este link no navegador:</p>
			<p>%s</p>
			<hr>
			<p style="color: #666; font-size: 12px;">Powered by eSimulate</p>
		</body>
		</html>
	`, logoHTML, companyName, candidateName, companyName, examURL, examURL)
	
	return e.SendEmail(to, fmt.Sprintf("Convite para Teste Técnico - %s", companyName), body)
}

// SendContactAdminEmail envia mensagem de contato para admin
func (e *EmailService) SendContactAdminEmail(senderEmail, subject, message string) error {
	adminEmail := getEnv("ADMIN_EMAIL", "admin@esimulate.com")
	
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<h2>Nova Mensagem de Contato</h2>
			<p><strong>De:</strong> %s</p>
			<p><strong>Assunto:</strong> %s</p>
			<hr>
			<p>%s</p>
			<hr>
			<p style="color: #666; font-size: 12px;">Sistema eSimulate</p>
		</body>
		</html>
	`, senderEmail, subject, template.HTMLEscapeString(message))
	
	return e.SendEmail(adminEmail, fmt.Sprintf("[Contato] %s", subject), body)
}

