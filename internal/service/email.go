package service

import (
	"crypto/tls"
	"esimulate-backend/internal/logger"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"time"
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

// dialSMTP faz a conexão SMTP com timeout customizado (para portas que usam STARTTLS)
func dialSMTP(addr string, timeout time.Duration) (*smtp.Client, error) {
	// Tentar primeiro com smtp.Dial (mais robusto)
	client, err := smtp.Dial(addr)
	if err == nil {
		return client, nil
	}
	
	// Se smtp.Dial falhar, tentar com conexão manual
	logger.Debug("[EMAIL] smtp.Dial falhou, tentando conexão manual | Erro: %v", err)
	
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar: %w", err)
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("endereço inválido: %w", err)
	}

	// Criar cliente com a conexão manual
	// smtp.NewClient lê a resposta inicial do servidor automaticamente
	client, err = smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		// Tentar ler qualquer mensagem de erro do servidor antes de fechar
		return nil, fmt.Errorf("falha ao criar cliente SMTP (servidor pode ter fechado a conexão): %w", err)
	}

	return client, nil
}

// dialSMTPSSL faz a conexão SMTP com SSL direto (para porta 465)
func dialSMTPSSL(addr string, timeout time.Duration) (*smtp.Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar: %w", err)
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("endereço inválido: %w", err)
	}

	// Configurar TLS direto
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
	}

	// Fazer upgrade para TLS
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("falha no handshake TLS: %w", err)
	}

	// Criar cliente SMTP sobre a conexão TLS
	client, err := smtp.NewClient(tlsConn, host)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("falha ao criar cliente SMTP: %w", err)
	}

	return client, nil
}

// SendEmail envia um email simples
func (e *EmailService) SendEmail(to, subject, body string) error {
	logger.Debug("[EMAIL] Iniciando envio | Para: %s | Assunto: %s", to, subject)
	
	// Se não houver configuração SMTP, apenas logar (modo desenvolvimento)
	if e.SMTPUser == "" || e.SMTPPassword == "" {
		logger.Info("[EMAIL SIMULADO] Enviado para: %s | Assunto: %s", to, subject)
		return nil
	}

	// Converter porta para inteiro
	port, err := strconv.Atoi(e.SMTPPort)
	if err != nil {
		logger.Error("[EMAIL] Porta inválida | Porta: %s | Erro: %v", e.SMTPPort, err)
		return fmt.Errorf("porta SMTP inválida: %s", e.SMTPPort)
	}

	// Montar mensagem
	msg := []byte(fmt.Sprintf("From: %s <%s>\r\n", e.FromName, e.FromEmail) +
		fmt.Sprintf("To: %s\r\n", to) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	// Conectar ao servidor SMTP com timeout
	addr := fmt.Sprintf("%s:%d", e.SMTPHost, port)
	logger.Debug("[EMAIL] Conectando ao SMTP | Endereço: %s", addr)
	
	var client *smtp.Client
	
	// Porta 465 usa SSL direto, outras portas usam STARTTLS
	if port == 465 {
		client, err = dialSMTPSSL(addr, 15*time.Second)
	} else {
		client, err = dialSMTP(addr, 15*time.Second)
	}
	
	if err != nil {
		logger.Error("[EMAIL] Erro ao conectar ao SMTP | Endereço: %s | Erro: %v", addr, err)
		return fmt.Errorf("falha ao conectar ao servidor SMTP: %w", err)
	}
	defer client.Close()

	// Verificar se o servidor suporta STARTTLS (apenas se não for porta 465 que já usa SSL)
	if port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{
				ServerName:         e.SMTPHost,
				InsecureSkipVerify: false,
			}
			if err := client.StartTLS(config); err != nil {
				logger.Error("[EMAIL] Erro ao iniciar STARTTLS | Erro: %v", err)
				return fmt.Errorf("falha ao iniciar STARTTLS: %w", err)
			}
		}
	}

	// Autenticar
	auth := smtp.PlainAuth("", e.SMTPUser, e.SMTPPassword, e.SMTPHost)
	if err := client.Auth(auth); err != nil {
		logger.Error("[EMAIL] Erro na autenticação | Erro: %v", err)
		return fmt.Errorf("falha na autenticação SMTP: %w", err)
	}

	// Definir remetente
	if err := client.Mail(e.FromEmail); err != nil {
		logger.Error("[EMAIL] Erro ao definir remetente | Erro: %v", err)
		return fmt.Errorf("falha ao definir remetente: %w", err)
	}

	// Definir destinatário
	if err := client.Rcpt(to); err != nil {
		logger.Error("[EMAIL] Erro ao definir destinatário | Para: %s | Erro: %v", to, err)
		return fmt.Errorf("falha ao definir destinatário: %w", err)
	}

	// Enviar dados
	writer, err := client.Data()
	if err != nil {
		logger.Error("[EMAIL] Erro ao obter writer | Erro: %v", err)
		return fmt.Errorf("falha ao obter writer de dados: %w", err)
	}

	_, err = writer.Write(msg)
	if err != nil {
		writer.Close()
		logger.Error("[EMAIL] Erro ao escrever mensagem | Erro: %v", err)
		return fmt.Errorf("falha ao escrever mensagem: %w", err)
	}

	if err := writer.Close(); err != nil {
		logger.Error("[EMAIL] Erro ao fechar writer | Erro: %v", err)
		return fmt.Errorf("falha ao fechar writer: %w", err)
	}

	// Encerrar conexão (não crítico se falhar)
	if err := client.Quit(); err != nil {
		logger.Debug("[EMAIL] Aviso ao encerrar conexão | Erro: %v", err)
	}
	
	logger.Info("[EMAIL] Enviado com sucesso | Para: %s | Assunto: %s", to, subject)
	return nil
}

// SendVerificationEmail envia email de verificação
func (e *EmailService) SendVerificationEmail(to, name, token string) error {
	logger.Debug("[EMAIL] Tipo: Verificação | Para: %s | Nome: %s", to, name)
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
	logger.Debug("[EMAIL] Tipo: Recuperação de Senha | Para: %s | Nome: %s", to, name)
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
	logger.Debug("[EMAIL] Tipo: Convite Empresa | Para: %s | Candidato: %s | Empresa: %s", 
		to, candidateName, companyName)
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
	logger.Info("[EMAIL] Tipo: Contato Admin | De: %s | Assunto: %s", senderEmail, subject)
	
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

